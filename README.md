# LifeBase

클라우드 + 캘린더 + Todo 통합 개인 플랫폼.

Google OAuth 단일 인증으로 파일 관리, 일정, 할 일을 하나의 서비스에서 관리한다.

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
│   │   │   ├── cloud/       # 파일/폴더 관리
│   │   │   ├── gallery/     # 갤러리 (썸네일 서빙)
│   │   │   ├── calendar/    # 캘린더 + 이벤트
│   │   │   ├── todo/        # Todo (계층/우선순위/고정)
│   │   │   ├── sharing/     # 폴더 공유 (초대 토큰 + ACL)
│   │   │   ├── settings/    # 사용자 설정
│   │   │   ├── worker/      # 썸네일 생성 워커
│   │   │   └── shared/      # 공통 (미들웨어, 설정, 응답)
│   │   └── migrations/  # goose DB 마이그레이션 (10개)
│   ├── web/             # Next.js 웹 앱
│   │   └── src/app/     # 페이지: Cloud, Gallery, Calendar, Todo, Settings
│   ├── desktop/         # Tauri v2 데스크탑 앱
│   │   └── src/         # Rust (main.rs, lib.rs)
│   └── mobile/          # Expo React Native 모바일 앱
│       └── app/         # 탭: Cloud, Gallery, Calendar, Todo, Settings
├── docs/                # 설계 문서 (16개)
├── packages/            # 공유 패키지
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
cp apps/server/.env.example apps/server/.env
# .env 파일에 Google OAuth 클라이언트 ID/Secret, DB URL 등 설정
```

### 서버 실행

```bash
cd apps/server

# DB 마이그레이션
goose -dir migrations postgres "$DATABASE_URL" up

# 서버 시작 (기본 포트: 38117)
go run ./cmd/server/
```

### 웹 실행

```bash
pnpm install
pnpm --filter @lifebase/web dev
# http://localhost:39001
```

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

### Cloud
- 파일 업로드/다운로드/삭제/이동/이름변경
- 폴더 CRUD + 계층 탐색
- 휴지통 (복원/비우기)
- 파일 검색 (pg_trgm)
- 정렬: 이름/크기/수정일/생성일

### Gallery
- 이미지/비디오 썸네일 자동 생성 (Asynq 비동기 워커)
- 격자/리스트/날짜별 뷰
- 미디어 타입 필터, 무한 스크롤
- EXIF 메타데이터 추출 (촬영일/GPS/카메라)

### Calendar
- 캘린더 CRUD + 이벤트 CRUD
- 리마인더 관리
- 월간/주간/3일/일정 4개 뷰

### Todo
- 리스트 기반 관리
- 2단계 계층 (부모-자식)
- 우선순위 4단계 (urgent/high/normal/low)
- 고정(Pin) 최대 5개

### 공유
- 10분 만료 초대 토큰
- ACL: viewer(읽기)/editor(수정)

### Settings
- 5탭: 일반/캘린더/Todo/알림/Cloud
- 테마 (라이트/다크/시스템)
- 방해 금지 시간 설정

## API

모든 API는 `/api/v1/` 프리픽스. 인증이 필요한 엔드포인트는 `Authorization: Bearer <token>` 헤더 필요.

| 모듈 | 엔드포인트 수 |
|------|-------------|
| auth | 5 |
| cloud | 12 |
| gallery | 2 |
| calendar | 8 |
| todo | 8 |
| settings | 2 |
| sharing | 5 |
| **합계** | **42** |

상세 목록은 `docs/700-마일스톤.md` 참조.

## 테스트

```bash
cd apps/server
go test ./...
# 14개 테스트 (Todo 5, Sharing 6, Response 3)
```

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
| `docs/220-할일-기능.md` | Todo 계층, 우선순위, 고정, Google Tasks |
| `docs/400-시스템-아키텍처.md` | 헥사고날 구조, 인프라, 파일 스토리지, 썸네일 |
| `docs/420-플랫폼-기술선정.md` | 스택 선정 이유, 플랫폼별 역할 |
| `docs/700-마일스톤.md` | 10단계 구현 로드맵 + 현재 진행 상황 |

전체 문서 인덱스: `plan.md`

## 버전

현재: **v0.1.0** (초기 구현 완료)

Semantic Versioning 사용. 단일 소스: `package.json`

## 라이선스

Private
