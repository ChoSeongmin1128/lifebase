package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	marshalThumbnailPayload = json.Marshal
	mkdirAll               = os.MkdirAll
	runThumbnailFn         = runVipsThumbnail
	execCommand            = exec.Command
	execThumbnailSQL       = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) error {
		_, err := db.Exec(ctx, sql, args...)
		return err
	}
)

type ThumbnailPayload struct {
	FileID      string `json:"file_id"`
	UserID      string `json:"user_id"`
	StoragePath string `json:"storage_path"`
	MimeType    string `json:"mime_type"`
}

func NewThumbnailTask(fileID, userID, storagePath, mimeType string) (*asynq.Task, error) {
	payload, err := marshalThumbnailPayload(ThumbnailPayload{
		FileID:      fileID,
		UserID:      userID,
		StoragePath: storagePath,
		MimeType:    mimeType,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TypeThumbnailGenerate, payload, asynq.MaxRetry(3)), nil
}

type ThumbnailHandler struct {
	db        *pgxpool.Pool
	dataPath  string
	thumbPath string
}

func NewThumbnailHandler(db *pgxpool.Pool, dataPath, thumbPath string) *ThumbnailHandler {
	return &ThumbnailHandler{db: db, dataPath: dataPath, thumbPath: thumbPath}
}

func (h *ThumbnailHandler) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var p ThumbnailPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}

	slog.Info("generating thumbnail", "file_id", p.FileID, "mime_type", p.MimeType)

	// Update status to processing
	if err := execThumbnailSQL(ctx, h.db,
		`UPDATE files SET thumb_status = 'processing' WHERE id = $1`, p.FileID); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	srcPath := filepath.Join(h.dataPath, p.StoragePath)
	thumbDir := filepath.Join(h.thumbPath, p.UserID)
	if err := mkdirAll(thumbDir, 0755); err != nil {
		return fmt.Errorf("create thumb dir: %w", err)
	}

	smallPath := filepath.Join(thumbDir, p.FileID+"_small.webp")
	mediumPath := filepath.Join(thumbDir, p.FileID+"_medium.webp")

	var err error
	if strings.HasPrefix(p.MimeType, "image/") {
		err = h.generateImageThumbnails(srcPath, smallPath, mediumPath)
	} else if strings.HasPrefix(p.MimeType, "video/") {
		err = h.generateVideoThumbnails(srcPath, smallPath, mediumPath)
	} else {
		// Not a media file, mark as done
		_ = execThumbnailSQL(ctx, h.db,
			`UPDATE files SET thumb_status = 'done' WHERE id = $1`, p.FileID)
		return nil
	}

	if err != nil {
		slog.Error("thumbnail generation failed", "file_id", p.FileID, "error", err)
		_ = execThumbnailSQL(ctx, h.db,
			`UPDATE files SET thumb_status = 'failed' WHERE id = $1`, p.FileID)
		return fmt.Errorf("generate thumbnails: %w", err)
	}

	// Extract EXIF taken_at for images
	if strings.HasPrefix(p.MimeType, "image/") {
		h.extractExifTakenAt(ctx, srcPath, p.FileID)
	}

	if err := execThumbnailSQL(ctx, h.db,
		`UPDATE files SET thumb_status = 'done' WHERE id = $1`, p.FileID); err != nil {
		return fmt.Errorf("update status done: %w", err)
	}

	slog.Info("thumbnail generated", "file_id", p.FileID)
	return nil
}

func (h *ThumbnailHandler) generateImageThumbnails(src, smallDst, mediumDst string) error {
	// small: 150x150
	if err := runThumbnailFn(src, smallDst, 150); err != nil {
		return fmt.Errorf("small thumbnail: %w", err)
	}
	// medium: 400x400
	if err := runThumbnailFn(src, mediumDst, 400); err != nil {
		return fmt.Errorf("medium thumbnail: %w", err)
	}
	return nil
}

func (h *ThumbnailHandler) generateVideoThumbnails(src, smallDst, mediumDst string) error {
	// Extract frame at 1 second
	tmpFrame := smallDst + ".tmp.png"
	defer os.Remove(tmpFrame)

	cmd := execCommand("ffmpeg", "-y", "-i", src, "-ss", "00:00:01", "-vframes", "1", "-q:v", "2", tmpFrame)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg extract frame: %w (%s)", err, string(out))
	}

	// Convert frame to thumbnails
	if err := runThumbnailFn(tmpFrame, smallDst, 150); err != nil {
		return fmt.Errorf("small video thumbnail: %w", err)
	}
	if err := runThumbnailFn(tmpFrame, mediumDst, 400); err != nil {
		return fmt.Errorf("medium video thumbnail: %w", err)
	}
	return nil
}

func runVipsThumbnail(src, dst string, size int) error {
	cmd := execCommand("vipsthumbnail", src,
		"--size", fmt.Sprintf("%dx%d", size, size),
		"--output", dst+"[Q=80]",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("vipsthumbnail: %w (%s)", err, string(out))
	}
	return nil
}

func (h *ThumbnailHandler) extractExifTakenAt(ctx context.Context, srcPath, fileID string) {
	// Use vipsheader to extract EXIF date
	cmd := execCommand("exiftool", "-DateTimeOriginal", "-s3", srcPath)
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return
	}

	dateStr := strings.TrimSpace(string(out))
	if dateStr == "" {
		return
	}

	// EXIF format: "2025:01:15 14:30:00"
	// Convert to ISO: "2025-01-15T14:30:00"
	dateStr = strings.Replace(dateStr, ":", "-", 2)
	dateStr = strings.Replace(dateStr, " ", "T", 1)

	_, _ = h.db.Exec(ctx,
		`UPDATE files SET taken_at = $2 WHERE id = $1 AND taken_at IS NULL`,
		fileID, dateStr)
}
