# LifeBase

클라우드 + 캘린더 + Todo 통합 개인 플랫폼.

Google OAuth 단일 인증으로 파일 관리, 일정, 할 일을 하나의 서비스에서 관리한다.

개발 협업 워크플로우는 `CONTRIBUTING.md`를 기준으로 운영한다.

## 스택

| 영역 | 기술 |
|------|------|
| 백엔드 | Go (chi), 헥사고날 아키텍처 |
| 데이터베이스 | PostgreSQL 17 |
| 캐시/큐 | Redis + Asynq |
| 웹 | Next.js 16 (App Router), Tailwind CSS |
| 데스크탑 | Tauri v2 (Rust + WebView) |
| 모바일 | React Native (Expo SDK 54) |
| 파일 저장소 | 로컬 파일시스템 (UUID 기반) |
| 미디어 처리 | libvips (이미지), FFmpeg (비디오) |
| 인프라 | Mac Mini bare metal, Caddy (HTTPS), Cloudflare DNS |

## 프로젝트 구조

```
lifebase/
├── apps/
│   ├── server/          # Go API 서버
│   │   ├── cmd/server/  # 엔트리포인트
│   │   ├── internal/    # 헥사고날 모듈
│   │   │   ├── auth/        # 인증 (Google OAuth + JWT)
│   │   │   ├── home/        # Home 요약 허브
│   │   │   ├── holiday/     # 한국 공휴일 조회/갱신
│   │   │   ├── cloud/       # 파일/폴더 관리
│   │   │   ├── gallery/     # 갤러리 (썸네일 서빙)
│   │   │   ├── calendar/    # 캘린더 + 이벤트
│   │   │   ├── todo/        # Todo (계층/기한/고정)
│   │   │   ├── sharing/     # 폴더 공유 (초대 토큰 + ACL)
│   │   │   ├── settings/    # 사용자 설정
│   │   │   ├── admin/       # 관리자 운영 (권한/할당량/계정 상태)
│   │   │   ├── worker/      # 썸네일 생성 워커
│   │   │   └── shared/      # 공통 (미들웨어, 설정, 응답)
│   │   └── migrations/  # goose DB 마이그레이션
│   ├── web/             # Next.js 웹 앱
│   │   └── src/app/     # 페이지: Home, Cloud, Gallery, Calendar, Todo, Settings, Admin
│   ├── desktop/         # Tauri v2 데스크탑 앱
│   │   └── src/         # Rust (main.rs, lib.rs)
│   └── mobile/          # Expo React Native 모바일 앱
│       └── app/         # 탭: Cloud, Gallery, Calendar, Todo, Settings
├── docs/                # 설계 문서 및 프로토타입
├── packages/            # 공유 패키지 (domain, api-types, design-tokens, features/*)
└── resources/           # 디자인 에셋
```

## 실행 방법

### 사전 요구사항

- Go 1.22+
- Node.js 20+ / pnpm 10+
- PostgreSQL 17
- Redis 8+
- libvips, FFmpeg (Gallery 썸네일)

### 환경 변수

```bash
# 프로젝트 루트 기준
# 로컬 개발 값은 .env.development.local에 저장 (git ignored)
# 운영 값은 .env.production.local에 저장 (git ignored)
# 웹 앱 값은 apps/web/.env.development.local, apps/web/.env.production.local 사용

# 기본 SERVER_ENV=development는 .env에 정의됨
```

Go 서버(`apps/server`)는 아래 우선순위로 환경 변수를 읽는다.

1. 프로세스 환경 변수(export)
2. `.env.<SERVER_ENV>.local`
3. `.env.local`
4. `.env.<SERVER_ENV>`
5. `.env`

`SERVER_ENV`가 없으면 기본값은 `development`다.

- 로컬 개발 권장: `SERVER_ENV=development` + `.env.development.local`
- 운영 권장: `SERVER_ENV=production` + `.env.production.local` (또는 프로세스 환경 변수)

### worktree bootstrap

새 worktree에서는 초기화 단계를 별도로 한 번 수행한다.

```bash
pnpm bootstrap:worktree
```

- 원본 worktree의 루트/apps/web 환경 파일을 현재 worktree로 복사한다.
- 현재 worktree에서 `pnpm install`을 실행한다.
- 이미 있는 환경 파일은 기본적으로 덮어쓰지 않는다.
- 덮어써야 하면 `pnpm bootstrap:worktree -- --force`를 사용한다.

### 로컬 통합 개발 실행

```bash
pnpm dev
pnpm dev:status
pnpm dev:stop
```

- `pnpm dev`는 API 서버와 Web dev 서버를 함께 백그라운드로 올린다.
- 로그는 `tmp/dev-stack/logs/server.log`, `tmp/dev-stack/logs/web.log`에 기록된다.
- 기본 포트는 API `38117`, Web `39001`이고, 이미 사용 중이면 빈 포트를 찾아 자동으로 올린다.
- `pnpm dev`는 현재 API 포트를 `NEXT_PUBLIC_API_URL`로 Web에 주입하므로, Web API 호출은 포트 자동 상승 상황에서도 같은 세션 안에서 유지된다.
- 포트가 기본값이 아니면 Google OAuth 로컬 redirect URI를 같은 포트로 맞춰야 로그인 흐름이 동작한다.
- 같은 worktree에서 이미 `next dev`를 수동으로 띄운 상태면 `.next/dev/lock` 때문에 `pnpm dev`가 실패하므로 먼저 기존 프로세스를 정리해야 한다.

### 서버 실행

```bash
cd apps/server

# DB 마이그레이션
goose -dir migrations postgres "$DATABASE_URL" up

# 서버 시작 (기본 포트: 38117)
go run ./cmd/server/

# 기존 스토리지/썸네일 루트의 빈 디렉터리 정리
go run ./cmd/cleanup-empty-dirs/
```

### 웹 실행

```bash
pnpm install
pnpm --filter @lifebase/web dev
# http://localhost:39001
```

- Web 클라이언트는 `NEXT_PUBLIC_API_URL`이 있으면 해당 origin을 직접 사용하고, 없으면 `/api/v1` 상대 경로를 사용한다.
- 개발 모드에서는 Next rewrite가 `/api/v1/*`를 `NEXT_PUBLIC_API_URL` 또는 기본 `http://localhost:38117`로 프록시한다.

### 데스크탑 실행 (macOS)

```bash
# Rust + Tauri CLI 필요
cargo install tauri-cli --version "^2"

cd apps/desktop
cargo tauri dev
# 웹 dev 서버(localhost:39001)가 먼저 실행되어 있어야 함
```

### 모바일 실행

```bash
cd apps/mobile
pnpm install
npx expo start
# Expo Go 앱 또는 시뮬레이터에서 실행
```

## 주요 기능

### Home
- 오늘 일정/지난 Todo/최근 파일/저장공간 요약
- 빠른 액션: 일정 추가, Todo 추가, 파일 업로드
- Web/Desktop Home은 인증 이후 공통 page shell 안에서 동작하고, 사이드바 보조 내비는 `?focus=summary|calendar|todo|files|storage`로 각 섹션에 직접 진입한다

### Cloud
- 파일 업로드/다운로드/삭제/이동/이름변경
- 원본 파일 실제 삭제 시 비어 있는 UUID prefix/user 디렉터리를 즉시 정리
- 폴더 CRUD + 계층 탐색
- Web Cloud는 `/cloud/folders/{folderId}`를 canonical 폴더 URL로 사용하고, 기존 `?folder=` 진입은 canonical route로 정리한다
- Web Cloud 폴더 화면은 상단 breadcrumb + 현재 폴더명 헤더를 기본으로 사용하고, 탐색 중 경로 전체가 새로고침되듯 흔들리지 않도록 로컬 경로 상태를 우선 사용한다
- Web Cloud 로딩 신호는 상단 헤더 한 곳으로만 모으고, 폴더 전환 중에는 지연된 작은 indicator만 보여 일관된 전환감을 유지한다
- Web Cloud는 잘못된 UUID, 삭제된 폴더 링크, 일시적 폴더 조회 실패를 빈 폴더로 위장하지 않고 invalid/not-found/error 상태로 분리해 보여준다
- Mobile/Desktop Cloud도 같은 정보 구조를 기준으로 후속 정렬한다
- Web/Desktop Cloud에서 더블클릭은 폴더 열기에만 사용하고, 파일 편집/다운로드는 컨텍스트 메뉴 또는 일괄 작업에서만 명시적으로 실행한다
- Web/Desktop Cloud에서 선택된 항목 중 하나를 폴더로 드래그하면 선택된 항목 전체를 함께 이동한다
- Web/Desktop Cloud 복사/이동 클립보드는 폴더 이동 후에도 유지되고, 선택 일괄 바에서 복사/이동/다운로드/삭제를 직접 실행할 수 있다
- Web/Desktop Cloud 붙여넣기(복사/이동)도 실행 직후 5초 실행 취소를 제공한다
- 휴지통 (복원/비우기)
- Web Cloud 파일/폴더 삭제는 우측 하단 Undo 토스트 5초를 제공하고, 시간 경과 후 휴지통 이동을 확정한다
- 폴더 삭제/복원/휴지통 비우기는 하위 폴더·파일까지 재귀로 함께 처리하고, 휴지통에서도 폴더 내부를 탐색할 수 있다
- Web/Desktop Cloud는 `cmd/ctrl+a`로 현재 보이는 항목 전체 선택을 지원하고, 클립보드는 다중 파일 `copy/cut/paste`, `Esc` 선택/클립보드 해제를 지원한다
- Web Cloud `새 파일` 기본 확장자는 `txt`를 사용하고, 확장자 토글도 `.txt`를 먼저 보여준 뒤 필요할 때만 `md`로 바꿔 생성한다
- 파일 검색 (pg_trgm)
- 정렬: 이름/크기/수정일/생성일
- Web/Mobile 공통 파일 타입 아이콘 색상 토큰 적용

### Gallery
- 이미지/비디오 썸네일 자동 생성 (Asynq 비동기 워커)
- 격자/리스트/날짜별 뷰
- 미디어 타입 필터, 무한 스크롤
- EXIF 메타데이터 추출 (촬영일/GPS/카메라)
- Web/Desktop Gallery는 공통 page shell을 사용하고, 뷰/미디어 타입/정렬 상태를 URL에 반영해 새로고침과 딥링크에서 같은 상태를 복원한다

### Calendar
- 캘린더 CRUD + 이벤트 CRUD
- 리마인더 관리
- 월간/주간/3일/일정 4개 뷰
- 다중 Google 계정 필터/색상 정책
- 한국 공휴일 overlay 표시 + 설정 토글
- Web Calendar 이벤트 삭제는 우측 하단 Undo 토스트 5초를 제공하고, 시간 경과 후 실제 삭제를 확정한다

### Todo
- 리스트 기반 관리
- 1단계 계층 (부모-자식)
- 고정(Pin) 최대 5개
- notes 미리보기 표시 (1~2줄)
- due 모델: `due_date`(필수 날짜) + `due_time`(선택 시간)
- Google Tasks 공개 API 제약상 동기화는 `due_date`만 왕복하고 `due_time`은 LifeBase 로컬 확장값으로 유지
- Google Tasks parent/reorder는 `tasks.move`로 반영하고, 1단계를 넘는 원격 parent 체인은 최상위 부모 기준으로 정규화
- 완료된 자식 Todo는 활성 부모 아래에서 계층을 유지해 Google Tasks와 같은 부모 체인으로 표시
- 기본 정렬: `due`
- 정렬: `manual` / `due` / `recent_starred` / `title`
- 로컬에서 삭제 요청한 Todo는 Google push 삭제가 끝날 때까지 background pull sync가 다시 복구하지 않는다
- Web Todo 단건 삭제와 완료 항목 정리는 우측 하단 Undo 토스트 5초를 제공한다
- Web Todo는 `전체`/목록 전환 시 기존 목록을 유지한 채 백그라운드 refresh로 갱신한다
- Web/Desktop Todo는 제목을 최대 3줄까지 노출하고, 확장된 행 자체에서 제목을 바로 수정한다
- Web/Desktop Todo 목록은 차분한 surface형 행을 기본으로 하고, 목록/기한 메타는 제목 아래 칩으로 정리하며 우측 액션은 hover·확장 상태에서만 최소 노출한다
- Web/Desktop Todo 보조 내비는 `?scope=all|due|starred|completed`를 사용해 같은 화면 문법 안에서 범위를 전환한다
- 부모/자식 Todo는 접기 없이 항상 표시하고, 체크박스 왼쪽 gutter와 행 세로 밀도는 더 촘촘하게 유지한다
- Web/Desktop Todo 최상단 툴바는 현재 목록 드롭다운을 제목처럼 사용하고, 목록 전환/생성/삭제를 같은 줄에서 처리한다
- 최상단 툴바의 현재 목록 드롭다운과 액션 버튼은 고정 폭을 유지해 목록 이름 길이에 따라 주변 컨트롤 위치가 흔들리지 않게 한다
- Todo 검색은 최상단 툴바의 고정된 두 번째 줄에 유지하고, 검색 폭은 과하게 늘리지 않으며 정렬/동기화 액션과 분리해 줄바꿈이 내용 길이에 따라 흔들리지 않게 한다
- 필터는 제거하고 검색 + 정렬만 유지한다
- 목록 삭제는 최상단 툴바의 더보기 메뉴 안에 두고 확인 다이얼로그를 거치게 한다
- 확장 영역은 메모와 일정만 같은 surface 안에서 보조 편집해 Google Tasks에 가까운 인라인 편집 흐름을 유지한다
- 선택된 Todo는 같은 행을 다시 누르거나 빈 영역/Esc를 누르면 접히고, 내부 날짜/시간 레이어를 조작하는 동안은 편집을 유지한다. 날짜는 커스텀 달력 팝오버, 시간은 같은 팝오버 안에서 30분 단위 목록 선택 또는 직접 입력으로 조정한다
- 다른 Todo 행을 누르면 현재 편집은 부드럽게 닫히고, 선택한 행 편집으로 자연스럽게 전환된다
- Todo 확장 편집은 즉시 펼쳐지지 않고, 높이/투명도 전환으로 열고 닫을 때 모두 부드럽게 전환된다
- Web Cloud/Gallery도 섹션·폴더·필터 전환 시 기존 목록을 유지한 채 백그라운드 refresh로 갱신한다

### 공유
- 10분 만료 초대 토큰
- ACL: viewer(읽기)/editor(수정)

### Settings
- Web/Desktop Settings는 공통 page shell 안에서 좌측 섹션 레일을 사용하고 `/settings/general`, `/settings/calendar`, `/settings/todo`, `/settings/notifications`, `/settings/cloud` route로 직접 진입한다
- Mobile Settings는 같은 섹션 구조를 상단 세그먼트 칩으로 번역한다
- 테마 (라이트/다크/시스템)
- 방해 금지 시간 설정
- Google 계정 연결/별칭/색상/동기화 설정

### Admin
- 관리자 전용 OAuth 로그인 (`/admin/auth/callback`)
- `app=admin` 로그인은 `admin_users.is_active=true` 계정에만 토큰 발급
- 사용자 목록/상세 조회, 스토리지 사용량 재계산/초기화
- 사용자 할당량 수정 (숫자 + 단위 입력, B/KB/MB/GB/TB 표시)
- 신규 사용자 기본 할당량은 15GB이며, 기존 사용자 할당량은 관리자 정책에 따라 유지/조정한다
- Google 계정 상태 제어 (정상/재인증 필요/해지)

## API

모든 API는 `/api/v1/` 프리픽스. 인증이 필요한 엔드포인트는 `Authorization: Bearer <token>` 헤더 필요.

| 모듈 | 엔드포인트 수 |
|------|-------------|
| shared | 1 |
| auth | 9 |
| home | 1 |
| holiday | 2 |
| cloud | 26 |
| gallery | 2 |
| calendar | 11 |
| todo | 10 |
| settings | 2 |
| sharing | 5 |
| admin | 10 |
| **합계** | **79** |

상세 목록은 `docs/700-마일스톤.md` 참조.

## 테스트

```bash
cd apps/server
LIFEBASE_TEST_DATABASE_URL='postgres://<user>@localhost:5432/lifebase_test?sslmode=disable' \
go test -p 1 ./... -coverprofile=/tmp/lifebase-cover.out
go tool cover -func=/tmp/lifebase-cover.out | tail -n 1
```

DB 분리 기준:
- 개발 서버: `DATABASE_URL=postgres://<user>@localhost:5432/lifebase_dev?sslmode=disable`
- 테스트 전용: `LIFEBASE_TEST_DATABASE_URL=postgres://<user>@localhost:5432/lifebase_test?sslmode=disable`
- `apps/server/internal/testutil/dbtest`는 `lifebase_test`가 아닌 DB를 테스트 대상으로 거부

DB 백업/복구:
```bash
cd apps/server
DATABASE_URL='postgres://<user>@localhost:5432/lifebase?sslmode=disable' pnpm backup:db
bash ../../scripts/backup-db.sh
pnpm restore:db -- --file /Volumes/WDRedPlus/LifeBase/backups/manual/<backup-file>.dump
```

- 수동 백업 기본 경로: `/Volumes/WDRedPlus/LifeBase/backups/manual`
- 자동 백업 경로: `/Volumes/WDRedPlus/LifeBase/backups/{hourly,daily,weekly}`
- 자동 백업 보관: `hourly 14개`, `daily 14개`, `weekly 8개`
- 자동 백업은 운영 DB `lifebase`만 대상으로 하고, `scripts/backup-db.sh`가 `SERVER_ENV=production` 기준으로 실행
- 자동 백업/최근 백업 체크를 실제 운영에서 돌릴 때는 placeholder `.env.production.local` 대신 실제 `DATABASE_URL=.../lifebase`를 process env 또는 실제 운영 env 파일로 주입해야 한다
- `scripts/check-recent-db-backup.sh`는 운영 DB 대상 파괴적 작업 전에 최근 6시간 이내 백업이 없으면 실패
- 복구는 `pg_restore --clean --if-exists` 기반이므로 대상 DB를 덮어쓴다
- macOS 자동화는 `resources/launchd/cc.lifebase.db-backup.plist`를 `~/Library/LaunchAgents/`로 복사해 `launchctl load`로 등록한다

백엔드 테스트 정책:
- 백엔드 변경은 항상 TDD(`Fail -> Pass -> Refactor`)로 진행
- 전체 직렬 실행 기준 테스트 커버리지 `100%` 유지
- Go 테스트는 대상 코드와 같은 디렉토리에 `*_test.go`로 배치
- 공통 테스트 도우미는 `apps/server/internal/testutil` 사용

## 아키텍처

### 헥사고날 (Ports & Adapters)

```
internal/<module>/
├── domain/           # 엔티티, 값 객체
├── port/in/          # 입력 포트 (유스케이스 인터페이스)
├── port/out/         # 출력 포트 (저장소 인터페이스)
├── usecase/          # 비즈니스 로직
└── adapter/
    ├── in/http/      # HTTP 핸들러
    └── out/postgres/ # DB 구현
```

### 서버 중심 원칙
- 권한, 용량, 공유, 동기화 충돌 정책은 서버가 최종 판단
- 기능 규칙은 서버에서 단일 관리 (플랫폼별 분기 금지)

## 문서

| 문서 | 내용 |
|------|------|
| `docs/110-제품-비전.md` | 제품 방향, 차별화 |
| `docs/200-핵심-기능.md` | 전체 기능 범위, UI 정책, 정렬 규칙 |
| `docs/210-캘린더-기능.md` | 캘린더 6뷰, 반복, 알림, 동기화 |
| `docs/220-할일-기능.md` | Todo 계층, 기한, 고정, Google Tasks |
| `docs/400-시스템-아키텍처.md` | 헥사고날 구조, 인프라, 파일 스토리지, 썸네일 |
| `docs/420-플랫폼-기술선정.md` | 스택 선정 이유, 플랫폼별 역할 |
| `CONTRIBUTING.md` | 브랜치 전략, worktree 기준, squash merge 운영 |
| `docs/700-마일스톤.md` | 구현 로드맵 + 현재 진행 상황 |

전체 문서 인덱스: `plan.md`

## 버전

현재: **v0.2.0**

Semantic Versioning 사용. 단일 소스: `package.json`

## 라이선스

Private
