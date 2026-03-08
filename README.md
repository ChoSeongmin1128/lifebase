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
│   │   │   ├── todo/        # Todo (계층/우선순위/고정)
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

### Cloud
- 파일 업로드/다운로드/삭제/이동/이름변경
- 폴더 CRUD + 계층 탐색
- Web Cloud는 `/cloud/folders/{folderId}`를 canonical 폴더 URL로 사용하고, 기존 `?folder=` 진입은 canonical route로 정리한다
- Web Cloud 폴더 화면은 `뒤로가기 + 현재 폴더명 + 경로 보기` 패턴을 기본 헤더로 사용한다
- Mobile/Desktop Cloud도 같은 정보 구조를 기준으로 후속 정렬한다
- 휴지통 (복원/비우기)
- 파일 검색 (pg_trgm)
- 정렬: 이름/크기/수정일/생성일
- Web/Mobile 공통 파일 타입 아이콘 색상 토큰 적용

### Gallery
- 이미지/비디오 썸네일 자동 생성 (Asynq 비동기 워커)
- 격자/리스트/날짜별 뷰
- 미디어 타입 필터, 무한 스크롤
- EXIF 메타데이터 추출 (촬영일/GPS/카메라)

### Calendar
- 캘린더 CRUD + 이벤트 CRUD
- 리마인더 관리
- 월간/주간/3일/일정 4개 뷰
- 다중 Google 계정 필터/색상 정책
- 한국 공휴일 overlay 표시 + 설정 토글

### Todo
- 리스트 기반 관리
- 2단계 계층 (부모-자식)
- 우선순위 4단계 (urgent/high/normal/low)
- 고정(Pin) 최대 5개
- notes 미리보기 표시 (1~2줄)
- due 모델: `due_date`(필수 날짜) + `due_time`(선택 시간)
- Google Tasks 공개 API 제약상 동기화는 `due_date`만 왕복하고 `due_time`은 LifeBase 로컬 확장값으로 유지
- 정렬: `manual` / `date` / `due` / `recent_starred` / `title`
- Web Todo는 `전체`/목록 전환 시 기존 목록을 유지한 채 백그라운드 refresh로 갱신한다
- Web/Desktop Todo는 제목을 최대 3줄까지 노출하고, 확장된 행 자체에서 제목을 바로 수정한다
- 확장 영역은 메모, 일정, 우선순위만 보조 편집해 Google Tasks에 가까운 인라인 편집 흐름을 유지한다
- Web Cloud/Gallery도 섹션·폴더·필터 전환 시 기존 목록을 유지한 채 백그라운드 refresh로 갱신한다

- 선택된 Todo는 같은 행을 다시 누르거나 빈 영역/Esc를 누르면 접히고, 날짜·시간 칩 전체 클릭으로 picker를 연다
### 공유
- 10분 만료 초대 토큰
- ACL: viewer(읽기)/editor(수정)

### Settings
- 5탭: 일반/캘린더/Todo/알림/Cloud
- 테마 (라이트/다크/시스템)
- 방해 금지 시간 설정
- Google 계정 연결/별칭/색상/동기화 설정

### Admin
- 관리자 전용 OAuth 로그인 (`/admin/auth/callback`)
- 사용자 목록/상세 조회, 스토리지 사용량 재계산/초기화
- 사용자 할당량 수정 (숫자 + 단위 입력, B/KB/MB/GB/TB 표시)
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
| `docs/220-할일-기능.md` | Todo 계층, 우선순위, 고정, Google Tasks |
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
