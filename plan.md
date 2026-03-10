# LifeBase 문서 인덱스

이 파일은 문서 진입점입니다.

## 1. 방향 정의
- `docs/110-제품-비전.md`

## 2. 핵심 요구사항
- `docs/200-핵심-기능.md`
- `docs/210-캘린더-기능.md`
- `docs/220-할일-기능.md`
- `docs/230-일정-관리-정책.md`
- `docs/300-핵심-사용자-플로우.md`

## 3. 구현 뼈대
- `docs/400-시스템-아키텍처.md` — 헥사고날 구조, 인프라, 배포, 파일 스토리지
- `docs/410-구글-연동-요건.md`
- `docs/420-플랫폼-기술선정.md` — 스택, bare metal, goose, APFS, Cloudflare DNS

## 4. UX/디자인
- `docs/500-와이어프레임.md`
- `docs/510-디자인-규칙.md`
- `docs/520-색상-체계.md`
- `docs/530-UX-UI-테마.md`
- `docs/540-플랫폼-통일성.md`
- 프로토타입:
  - `docs/prototypes/calendar-year-view.html` — Year View (세로 바 + 기간 일정)
  - `docs/prototypes/calendar-palette.html` — 팔레트 비교 (월간 캘린더)
  - `docs/prototypes/lifebase-web-view.html` — Web 탭형 레이아웃
  - `docs/prototypes/lifebase-desktop-app-view.html` — Desktop App (폴더 동기화 포함)
  - `docs/prototypes/lifebase-mobile-view.html` — Mobile 탭형 레이아웃

## 5. 보안
- `docs/600-인증-OAuth.md`
- 2026-03-11 기준 Web/Admin은 httpOnly 쿠키 세션을 사용하고, OAuth `state` 검증은 callback/link 모두 필수다.

## 6. 실행 계획
- `docs/700-마일스톤.md`

## 7. 작업 규칙
- `CLAUDE.md`

## 7-1. 로컬 개발/worktree 실행 규칙
- 기본 작업 위치는 현재 repo의 `dev` 컨텍스트다.
- 사용자가 worktree를 명시적으로 요구한 경우에만 새 worktree를 만들고, 그때 `pnpm bootstrap:worktree`로 env 복사와 `pnpm install`을 먼저 수행한다.
- 로컬 통합 실행은 루트에서 `pnpm dev`를 사용한다.
- 실행 상태 확인은 `pnpm dev:status`, 종료는 `pnpm dev:stop`을 사용한다.
- 기본 포트는 API `38117`, Web `39001`이며, 충돌 시 빈 포트로 자동 조정될 수 있다.
- `pnpm dev`는 현재 API origin을 `NEXT_PUBLIC_API_URL`로 Web에 주입한다.
- Web API 호출은 `NEXT_PUBLIC_API_URL`이 없으면 `/api/v1` 상대 경로를 사용하고, 개발 모드에서는 Next rewrite로 로컬 API에 프록시한다.
- OAuth 로컬 로그인 검증이 필요하면 자동 조정된 포트와 Google redirect URI 구성이 일치하는지 확인한다.

## 8. 프론트 기능 구조 전환 현황 (진행중)
1. 목표
웹/모바일의 구현된 모든 프론트 기능을 화면 중심 구현에서 기능 중심 구조로 전환한다.

2. 현재 상태
- Web: `apps/web/src/features` 아래 `auth`, `admin`, `cloud`, `gallery`, `calendar`, `home`, `settings`, `todo`가 기능 단위 구조로 정리돼 있다.
- Admin Web: 로그인 성공 후에도 `admin_users.is_active=true` 검증을 통과한 계정만 admin 세션 유지 대상으로 본다.
- Web/Admin 프론트는 민감 JWT를 `localStorage`에 저장하지 않고, 브라우저에는 세션 마커만 남긴다.
- Mobile: `apps/mobile/features` 아래 `auth`, `cloud`, `gallery`, `calendar`, `todo`가 기능 단위 구조로 정리돼 있다.
- 공통 패키지: `packages/features/todo`가 존재하며, 나머지 기능은 앱 내부 feature 구조를 우선 사용 중이다.
- UI 토큰: `packages/design-tokens`에서 Cloud 파일 타입 색상/라벨 토큰을 Web/Mobile 공통으로 관리하기 시작했다.
- 서버 운영 안전장치: `lifebase_dev`/`lifebase_test` DB 분리와 `pg_dump`/`pg_restore` 기반 백업 커맨드를 도입했고, 운영 DB `lifebase`는 실제 `DATABASE_URL` 주입을 전제로 6시간/일간/주간 자동 백업과 최근 백업 차단 규칙을 사용한다.
- 서버 보안 기본값: `JWT_SECRET`, `STATE_HMAC_KEY`가 비어 있거나 개발용 기본값이면 서버가 시작되지 않는다.

3. 목표 구조
`packages/features/<feature>`에 `domain`, `usecase`, `repository`를 배치한다.

4. 앱별 어댑터 구조
웹/모바일은 각 앱 내부 `infrastructure`와 `ui/hooks`에서 공통 유스케이스를 연결한다.

5. 대상 기능
`auth`, `admin`, `cloud`, `gallery`, `calendar`, `todo`, `settings`를 전환 대상으로 한다.

6. 점진 마이그레이션 순서
1) 기능별 대표 흐름 선정
2) 응답 매핑 로직을 저장소 어댑터로 분리
3) 비즈니스 로직을 유스케이스로 추출
4) 유스케이스 테스트 보강
5) UI 페이지에서 훅으로 연결

7. 현재 우선순위
Todo 공통 패키지 패턴을 유지하되, Home/Calendar/Cloud처럼 플랫폼별 UI 차이가 큰 기능은 앱 내부 feature 구조를 먼저 안정화한 뒤 공유 범위를 좁혀 확장한다.

8. 인증 이후 공통 UI 셸 기준 (2026-03-09)
- Web/Desktop은 `sidebar + page header + content body + optional secondary panel` 구조를 공통 셸로 사용한다.
- 사이드바 보조 내비는 Home/Cloud/Calendar/Todo/Gallery/Settings 전체에 적용하고, 페이지별 상태와 URL을 우선적으로 맞춘다.
- Page header는 동일한 높이와 좌우 패딩, 동일한 border rhythm을 사용한다.
- surface는 `background / surface / muted surface / selected surface / panel` 계층으로 단순화하고, 강한 그림자 대신 border와 약한 배경 차이로 구분한다.
- Todo/Cloud/Gallery/Home은 같은 workspace 톤 안에서 표현을 달리하고, Settings는 같은 셸 안의 폼 화면 유형으로 취급한다.
- Mobile은 같은 정보 구조를 상단 세그먼트/칩으로 번역하고, Home은 이번 범위에서 제외한다.

9. Web Todo 로딩 UX
- 첫 진입에서만 전체 로딩 상태를 표시한다.
- `전체`/개별 목록 전환 시 기존 Todo 목록은 유지하고 백그라운드 refresh indicator만 노출한다.
- Web/Desktop Todo 수정은 선택한 행 자체에서 제목을 바로 편집하고, 행 아래 확장에서 메모/일정만 보조 편집한다.
- Todo 기본 행은 차분한 surface형 카드와 제목 하단 메타 칩 구조를 사용하고, 우측 액션은 hover·확장 상태에서만 최소 노출한다.
- Todo parent/child는 접기 토글 없이 항상 표시하고, 체크박스 왼쪽 gutter와 행 세로 밀도는 더 촘촘하게 유지한다.
- Todo 목록 전환은 별도 사이드바 대신 최상단 툴바의 현재 목록 드롭다운으로 처리하고, 목록 생성/삭제도 같은 줄에서 다룬다.
- 최상단 툴바의 현재 목록 드롭다운과 액션 버튼은 고정 폭을 유지해 목록 이름 길이에 따른 좌우 흔들림을 막는다.
- Todo 검색은 최상단 툴바의 고정된 두 번째 줄에 두고, 검색 폭은 과하게 늘리지 않으며 정렬/동기화 액션과 분리해 목록 이름이나 버튼 길이에 따라 줄 위치가 흔들리지 않게 한다.
- Todo 필터는 제거하고 검색 + 정렬만 유지한다.
- 목록 삭제는 최상단 툴바의 더보기 메뉴 안에 두고 확인 다이얼로그를 거치게 한다.
- 선택된 Todo는 같은 행 재클릭, 빈 영역 클릭, `Esc`로 닫되 내부 날짜/시간 레이어를 조작하는 동안은 편집을 유지한다. 날짜는 커스텀 달력 팝오버, 시간은 같은 팝오버 안에서 30분 단위 목록 선택 또는 직접 입력으로 조정한다.
- 다른 Todo 행 클릭은 바깥 클릭 닫힘으로 처리하지 않고, 현재 편집에서 선택한 행 편집으로 전환한다.
- 확장 편집은 높이/투명도 전환으로 열고 닫을 때 모두 부드럽게 전환되는 것을 기본으로 한다.
- Todo 행 제목은 최대 3줄까지 노출하고 notes는 보조 정보로 축소한다.

10. Web Cloud/Gallery 로딩 UX
- 첫 진입에서만 전체 로딩 상태를 표시한다.
- Cloud 섹션/폴더/정렬 전환과 Gallery 필터/정렬 전환 시 기존 목록은 유지하고 백그라운드 refresh indicator만 노출한다.

11. Web Cloud canonical UX
- Web Cloud 폴더 탐색은 `/cloud/folders/{folderId}`를 canonical route로 사용한다.
- 기존 `/cloud?folder={id}` 링크는 지원 종료가 아니라 canonical route로 정리하는 compatibility redirect 대상으로 유지한다.
- 폴더 내부 상단은 `Cloud` 공통 제목 아래 현재 폴더명 영역에서 경로를 함께 보여주고, 마지막 폴더명에서 현재 폴더 액션을 열 수 있게 유지한다.
- 루트 섹션과 경로 시작점은 모두 `내 드라이브`로 통일하고, 현재 폴더명 영역 안에서 경로를 함께 보여줘 서비스명/섹션명/경로명이 같은 단어로 과하게 갈라지지 않게 유지한다.
- Cloud 로딩 신호는 상단 헤더 한 곳으로만 두고, 짧은 이동에는 spinner를 지연 표시해 전환 감각을 일관되게 유지한다.
- Cloud 검색창은 실제 검색이 가능한 `내 드라이브`에서만 노출하고, 나머지 섹션에서는 숨겨 no-op UI를 제거한다.
- 현재 폴더 삭제는 경로 헤더에서 바로 실행하되, 삭제 직후 상위 폴더로 선이동하고 같은 5초 Undo 토스트로 원래 경로 복귀를 지원한다.
- 잘못된 UUID, 삭제된 폴더 링크, 일시적 폴더 조회 실패는 `빈 폴더`가 아니라 invalid/not-found/error 상태로 구분해 렌더링한다.
- Cloud `새 파일` 입력의 기본 확장자는 `txt`로 두고, 확장자 토글은 `.txt`를 먼저 보여주며 입력을 다시 열어도 같은 기본값으로 복원한다.
- Web/Desktop Cloud에서 더블클릭은 폴더 열기에만 사용하고, 파일 편집/다운로드는 컨텍스트 메뉴/일괄 작업에서만 허용한다.
- Web/Desktop Cloud에서 선택된 항목 중 하나를 폴더로 드래그하면 선택된 항목 전체를 함께 이동한다.
- Web Cloud 업로드는 우하단 persistent panel에서 파일별/전체 진행률, 취소, 실패 재시도를 다루고, 실행 취소 토스트와 같은 시각 패밀리를 사용하되 별도 작업 패널로 유지한다.
- Web/Desktop Cloud는 리스트/그리드에서 빈 배경 drag로 영역 선택을 지원하고, 선택 집합 drag는 첫 항목 이름 + 수량 배지를 보여주는 custom drag image를 사용한다.
- Web/Desktop Cloud 다중 선택 액션은 별도 배너를 추가하지 않고 기존 툴바 슬롯을 교체해 드래그 중 목록 위치가 흔들리지 않게 유지한다.
- Web Cloud는 폴더 드래그 미지원을 즉시 안내하고, 업로드 실패 시 multipart 임시 파일과 저장 파일 롤백 경로를 정리한다.
- Web/Desktop Cloud 파일 목록 본문은 기본 텍스트 선택을 막고, 이름 변경 입력처럼 실제 편집이 필요한 요소만 텍스트 선택을 허용한다.
- Web/Desktop Cloud 영역 선택 박스는 현재 보이는 scroll 범위까지만 그려지고, 더 바깥 선택은 auto-scroll로만 이어지게 유지한다.
- Web/Desktop Cloud 스크롤 본문은 좌우 내부 여백과 하단 selection gutter를 유지해 항목이 많아도 빈 캔버스에서 영역 선택을 시작할 수 있게 한다.
- Web Cloud 섹션 전환 시 이전 섹션 목록을 즉시 비워 `내 드라이브` 내용이 `휴지통`에 남아 보이는 상태를 방지한다.
- Web Cloud 휴지통 선택 바는 선택 항목 `복원/삭제` 전용으로만 쓰고, 전체 `휴지통 비우기`는 비선택 상태의 전역 액션으로만 노출한다.
- Web Cloud `휴지통 비우기`는 휴지통 루트에 비울 항목이 있을 때만 활성화해 빈 휴지통에서 no-op 실행 피드백이 나오지 않게 유지한다.
- Web Cloud 새로고침은 대기 중인 삭제/휴지통 비우기 undo 작업을 먼저 확정해 목록과 서버 상태가 어긋나지 않게 유지한다.
- Web/Desktop Cloud 복사/이동 클립보드는 폴더 이동 후에도 유지되고, 선택 일괄 바에서 복사/이동/다운로드/삭제를 직접 실행한다.
- Web/Desktop Cloud 붙여넣기(복사/이동)는 실행 직후 5초 실행 취소를 제공하고, Undo는 서버 발급 단기 `undo_token` 과 현재 항목 상태 일치 검증으로만 허용한다.
- Cloud 영구 삭제/붙여넣기 취소는 원본 파일과 썸네일을 함께 정리하고, Gallery 썸네일은 활성 파일 확인 뒤에만 노출한다.
- Gallery 썸네일 응답은 인증 사용자 전용 `private, no-store` 캐시 정책으로 내려 공유 캐시 재사용을 막는다.
- Mobile/Desktop Cloud는 같은 정보 구조를 후속 기준으로 맞춘다.

12. 확장 순서
2차 `calendar`, 3차 `cloud`, 4차 `settings`, 5차 `gallery`, 6차 `auth/admin` 순으로 전환한다.

13. 검증
웹 lint, 모바일 타입 검사, 기능 패키지 빌드/테스트를 수행해 회귀를 확인한다.

## 14. Web/Desktop 보조 내비 및 Settings 라우팅 현황
1. 공통 보조 내비
Web/Desktop 사이드바는 Home/Cloud/Calendar/Todo/Gallery/Settings 모두 서비스별 보조 내비를 같은 위치에 확장한다.

2. URL 반영 규칙
- Home: `?focus=summary|calendar|todo|files|storage`
- Todo: `?scope=all|due|starred|completed`
- Gallery: `?view=grid|list|timeline`, `?media=all|image|video`, `?sort=createdAt|takenAt|name`, `?order=asc|desc`
- Settings: `/settings/{section}`

3. 목적
보조 내비 선택 상태와 페이지 내부 상태를 일치시켜 새로고침, 딥링크, Desktop 래핑 환경에서 같은 문맥을 복원한다.

4. Settings 구조
Web/Desktop Settings는 좌측 섹션 레일, Mobile Settings는 상단 세그먼트 칩을 사용하되 섹션 의미는 동일하게 유지한다.

5. 남은 작업
Calendar의 보기 전환과 계정 필터도 같은 수준의 URL 명시성을 어디까지 강제할지 후속 결정이 필요하다.

## 15. Cloud 항목 드래그 이동 계획 (웹 1차 확장)
1. 범위
`내 드라이브` 섹션에서 파일과 폴더를 폴더로 드래그해 이동하는 기능을 적용한다.

2. 구현 레이어
`CloudRepository`-`ManageCloudUseCase`-`useCloudActions`-`cloud/page.tsx` 순으로 `moveFile`/`moveFolder` 흐름을 연결한다.

3. 상호작용 규칙
파일/폴더 드래그를 허용하고 드롭 타깃은 폴더로 제한한다.
선택된 항목 중 하나를 드래그하면 현재 선택된 항목 전체를 같은 대상 폴더로 이동한다.

4. 피드백 규칙
드롭 가능한 폴더에는 hover 강조를 적용하고 이동 중에는 중복 요청을 차단한다.

5. 예외 처리
같은 상위 경로로의 이동과 자기 자신 폴더로의 드롭은 무시하고 실패 시 콘솔 오류와 사용자 메시지로 원인을 알린다.

6. 검증
웹 타입체크/린트와 서버 테스트를 실행해 회귀를 확인한다.

## 16. Cloud 클립보드 액션 계획 (웹/데스크톱 2차)
1. 범위
Cloud 컨텍스트 메뉴에 복사/잘라내기/붙여넣기 액션을 추가하고 단축키를 연결한다.

2. 메뉴 위치
`more-vertical` 메뉴 상단에 복사/잘라내기/붙여넣기, 하단에 이름 변경/이동/다운로드/삭제를 배치한다.

3. 단축키
mac은 `cmd+a/x/c/v`, windows는 `ctrl+a/x/c/v`를 지원한다.

4. 가드 규칙
입력창/텍스트 에디터 포커스 상태에서는 단축키를 가로채지 않고 기본 동작을 유지한다.

5. 활성 범위
`내 드라이브`에서만 클립보드 액션을 활성화하고 최근/공유됨/중요/휴지통에서는 비활성 처리한다.

6. 복사 정책
파일 복사만 지원하고 폴더 복사는 미지원으로 고정한다.

7. 다중 선택/해제 규칙
다중 선택 파일은 `copy/cut/paste` 대상으로 함께 처리하고, `Esc`는 선택 상태와 클립보드 상태를 함께 해제한다.

## 17. Home 허브 현황 및 후속 확장
1. 현재 상태
Web은 `GET /api/v1/home/summary`를 사용해 오늘 일정, 지난/오늘 Todo, 최근 파일, 저장공간, 빠른 액션을 표시한다.

2. 구현 완료 범위
- 로그인 후 기본 진입을 `/home`으로 고정
- 사이드바 로고(`LB`) 클릭 시 Home 이동
- 사이드바 보조 내비는 `?focus=summary|calendar|todo|files|storage`를 사용해 섹션 위치를 직접 복원한다
- 빠른 액션 딥링크 연결
- 저장공간 카드와 파일 타입별 breakdown 표시
- 신규 사용자 기본 스토리지 할당량은 15GB를 기준으로 운영하고, 기존 사용자 할당량은 유지한다

3. 후속 작업
- Desktop/Mobile 전용 Home 화면 확장
- 카드별 drill-down UX와 새로고침/에러 상태 정교화

4. 빠른 액션 연결
- `/calendar?quick=create`
- `/todo?quick=create`
- `/cloud?quick=upload`

5. 저장공간 시각화
저장공간 카드는 원형 도넛형 바를 사용하고 파일 타입별 사용량을 이미지/비디오/기타로 분해해 퍼센트와 용량을 함께 표시한다.

## 18. 캘린더 다중 계정 필터/색상 전략 현황
1. 현재 상태
서버에는 다중 Google 계정 조회/연결/수동 sync API가 있고, Web Calendar와 Settings에는 계정 필터/색상/동기화 토글 UI가 구현돼 있다.

2. 색상 규칙
단일 계정 선택 시 Google 색상(`event.color_id -> calendar.color_id`)을 사용하고 다중 계정 동시 선택 시 계정 단위 통일 색상을 사용한다.

3. 계정 필터 규칙
캘린더 뷰 전체(Month/Week/3-Day/Agenda/Year)에 동일한 계정 필터를 적용한다.

4. 계정 선택 기본값
활성 Google 계정 전체 선택을 기본으로 하고 선택 상태는 사용자 설정으로 저장한다.

5. 백엔드 데이터 축
`calendars.google_account_id` 컬럼을 추가해 캘린더-계정 소속 관계를 명시적으로 관리한다. `todo_lists.google_account_id`와 sync 상태 테이블도 추가됐다.

6. 인증 API 확장
일반 사용자 대상 `GET /auth/google-accounts`, `POST /auth/google-accounts/link`, 수동 sync API를 통해 다중 계정 조회/연결/동기화를 지원한다.

7. 웹 UX 확장
Settings > 일반에서 Google 계정 목록과 추가 연결 액션을 제공하고 캘린더 툴바에서 계정 멀티 선택을 지원한다.

8. 남은 작업
- 계정별 폴링 워커 안정화
- Todo 쓰기 경로 Google Tasks 우선 전환 잔여 정리 (`tasks.move` parent/reorder 반영 완료, 계정 경계 예외 처리 잔여)
- 계정 경계 reorder/move 정책 마무리
- Web Todo 표시 계층을 Google Tasks와 동일하게 유지하는 후속 검증 완료

## 19. 다중 계정 특수 캘린더 + Todo 완료 동기화/출처 표기 통합 계획 (부분 폐기)
1. 유지 항목
Todo 완료 동기화, Todo 출처 표기, 완료 보존 정책은 유지한다.

2. 폐기 항목
특수 캘린더(휴일/생일) 선택/중복 병합 정책은 폐기하고, 공휴일 정책은 `23. KASI 공휴일 표시 현황`을 따른다.

## 20. Todo due 모델 및 정렬 기준 현황
1. due 저장 모델
Todo 기한은 `due_date`(날짜)와 선택 `due_time`(시/분)으로 관리한다. `due_time`은 `due_date` 없이 존재할 수 없다.

2. Google Tasks 제약
Google Tasks 공개 API는 `due`를 RFC3339 문자열로 반환하지만 실제 의미는 date-only다. 따라서 Google 동기화는 `due_date`만 왕복하고 `due_time`은 LifeBase 로컬 확장값으로 유지한다.

3. 표시 규칙
리스트 행에서는 제목 아래 notes를 1~2줄 미리보기로 노출한다. due는 `due_date`만 있으면 날짜만, `due_date + due_time`이면 시/분까지 표시한다.

4. 정렬 기준
Todo 정렬은 `manual`, `due`, `recent_starred`, `title` 4종으로 고정한다. 기본 정렬은 `due`다.

5. 정렬 의미
- `manual`: 저장된 `sort_order`
- `due`: `due_date ASC`, 같은 날짜는 `due_time NULLS LAST` 후 `due_time ASC`
- `recent_starred`: `starred_at DESC`
- `title`: locale-aware ASC

7. 삭제 동기화 보호
로컬에서 삭제 요청해 push outbox에 `delete`가 대기 중인 Todo와 그 직계 자식은 background pull sync가 다시 활성화하지 않는다.

8. 삭제 Undo UX
Web Todo 단건 삭제와 완료 항목 일괄 삭제는 우측 하단 토스트에서 5초간 `실행 취소`를 제공하고, 시간 경과 후 실제 삭제 API를 호출한다.

6. due 없는 항목 처리
`due` 정렬에서 `due_date`가 없는 Todo는 항상 뒤로 보낸다.

## 21. 삭제 Undo UX 확장 현황
1. 현재 상태
복구 가능한 삭제는 Web 기준으로 우측 하단 Undo 토스트 5초를 먼저 노출하고, 시간 경과 후 실제 삭제를 확정한다.

2. Todo
단건 삭제와 완료 항목 일괄 삭제는 `삭제됨 + 실행 취소`, Undo 시 `복원됨` 패턴으로 통일한다.

3. Calendar
이벤트 단건 삭제도 같은 5초 Undo 토스트를 사용하고, Undo를 누르면 즉시 복원한다.

4. Cloud
`내 드라이브`의 파일/폴더 삭제는 휴지통 이동 전에 같은 5초 Undo 토스트를 제공한다.
원본 파일 실제 삭제 시 비어 있는 UUID prefix/user 디렉터리를 즉시 정리하고, 비미디어 파일은 빈 썸네일 디렉터리를 만들지 않는다.

5. 예외
휴지통 비우기처럼 복구가 어려운 파괴적 액션은 Undo 대신 확인 다이얼로그 대상으로 유지한다.

6. 휴지통 계층 규칙
Cloud 폴더 삭제는 폴더 메타데이터만 따로 휴지통에 보내지 않고, 하위 폴더/파일 전체를 재귀 soft delete 한다. 휴지통은 root 목록과 폴더 내부 direct child 목록을 모두 탐색할 수 있어야 하며, 폴더 복원/비우기도 같은 재귀 범위를 따른다.

## 22. Google Calendar API 에러 매핑 표준화 계획
1. 범위
Google Calendar/Tasks 연동 중 발생하는 API 오류를 서버 공통 규칙으로 분류한다.

2. 공통 분류
`HTTP code + reason` 조합을 `재시도 여부`, `재인증 필요 여부`, `full sync 필요 여부`, `사용자 메시지`로 매핑한다.

3. 적용 지점
`google_syncer`, `google_sync_coordinator`, `google_push_processor`에 동일 정책을 적용한다.

4. 핵심 정책
401/403 인증 오류는 `reauth_required`로 전환하고, 410 syncToken 만료 계열은 full sync로 복구한다.

5. 재시도 정책
403/429 rate-limit, 409 conflict, 412 conditionNotMet, 5xx는 지수 백오프 재시도 대상으로 고정한다.

6. 문서 정책
`docs/410-구글-연동-요건.md`에 운영 기준 매핑표를 유지한다.

7. 검증
분류 유닛 테스트와 서버 전체 테스트를 수행하고, 로그/DB 상태(`google_sync_state`, `google_push_outbox`)를 점검한다.

## 23. KASI 공휴일 표시 현황
1. 데이터 소스
한국 공휴일은 한국천문연구원 `getRestDeInfo`만 사용한다.

2. API
인증 사용자용 `GET /holidays?start=YYYY-MM-DD&end=YYYY-MM-DD`, 관리자용 `POST /admin/holidays/refresh`를 제공한다.

3. 캐시 전략
월 단위 DB 캐시(`public_holidays_kr`)와 월 동기화 상태(`public_holiday_sync_state`)를 사용하고 신선도 TTL은 3시간으로 고정한다.

4. 백그라운드 갱신
서버 시작 직후 1회 warm-up 후 3시간 주기로 당해년±2년 범위를 자동 최신화한다.

5. 동시성 정책
월 단위 advisory lock으로 on-demand 조회, 백그라운드 작업, admin 수동 최신화의 중복 실행을 차단한다.

6. 캘린더 렌더링 정책
공휴일은 이벤트 객체가 아닌 overlay로 표시하고, 날짜 숫자/라벨과 공휴일명을 빨간색으로 렌더한다.

7. 사용자 설정
`calendar_show_public_holidays` 키로 표시 여부를 제어하며 기본값은 `true`로 해석한다.

8. Google 특수 캘린더 정책
Google holiday/birthday 캘린더는 동기화 대상에서 제외하고, 기존 특수 캘린더 데이터는 동기화 시 정리한다.

9. 운영 기능
Admin 페이지에 `공휴일 데이터 최신화` 버튼을 추가하고 기본 실행 범위는 당해년±2년으로 고정한다.

10. 현재 상태와 후속 작업
- Web Calendar와 Settings의 공휴일 표시/토글은 구현 완료
- Desktop는 Web 앱을 재사용하므로 동일 반영 범위를 따른다
- Mobile 전용 표시 UX와 운영 자동화 점검은 후속 범위다
