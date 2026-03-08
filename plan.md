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
- Mobile: `apps/mobile/features` 아래 `auth`, `cloud`, `gallery`, `calendar`, `todo`가 기능 단위 구조로 정리돼 있다.
- 공통 패키지: `packages/features/todo`가 존재하며, 나머지 기능은 앱 내부 feature 구조를 우선 사용 중이다.
- UI 토큰: `packages/design-tokens`에서 Cloud 파일 타입 색상/라벨 토큰을 Web/Mobile 공통으로 관리하기 시작했다.

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

10. Web Todo 로딩 UX
- 첫 진입에서만 전체 로딩 상태를 표시한다.
- `전체`/개별 목록 전환 시 기존 Todo 목록은 유지하고 백그라운드 refresh indicator만 노출한다.
- Web/Desktop Todo 수정은 선택한 행 자체에서 제목을 바로 편집하고, 행 아래 확장에서 메모/일정/우선순위를 보조 편집한다.
- 선택된 Todo는 같은 행 재클릭, 빈 영역 클릭, `Esc`로 닫되 내부 날짜/시간/우선순위 레이어를 조작하는 동안은 편집을 유지한다. 날짜는 커스텀 달력 팝오버, 시간은 같은 팝오버 안에서 30분 단위 목록 선택 또는 직접 입력, 우선순위는 컴팩트 팝오버에서 조정한다.
- 다른 Todo 행 클릭은 바깥 클릭 닫힘으로 처리하지 않고, 현재 편집에서 선택한 행 편집으로 전환한다.
- 확장 편집은 높이/투명도 전환으로 열고 닫을 때 모두 부드럽게 전환되는 것을 기본으로 한다.
- Todo 행 제목은 최대 3줄까지 노출하고 notes는 보조 정보로 축소한다.

11. Web Cloud/Gallery 로딩 UX
- 첫 진입에서만 전체 로딩 상태를 표시한다.
- Cloud 섹션/폴더/정렬 전환과 Gallery 필터/정렬 전환 시 기존 목록은 유지하고 백그라운드 refresh indicator만 노출한다.

12. Web Cloud canonical UX
- Web Cloud 폴더 탐색은 `/cloud/folders/{folderId}`를 canonical route로 사용한다.
- 기존 `/cloud?folder={id}` 링크는 지원 종료가 아니라 canonical route로 정리하는 compatibility redirect 대상으로 유지한다.
- 폴더 내부 상단은 slash breadcrumb 대신 `뒤로가기 + 현재 폴더명 + 경로 보기` 패턴을 사용한다.
- Mobile/Desktop Cloud는 같은 정보 구조를 후속 기준으로 맞춘다.

8. 확장 순서
2차 `calendar`, 3차 `cloud`, 4차 `settings`, 5차 `gallery`, 6차 `auth/admin` 순으로 전환한다.

9. 검증
웹 lint, 모바일 타입 검사, 기능 패키지 빌드/테스트를 수행해 회귀를 확인한다.

## 9. Cloud 항목 드래그 이동 계획 (웹 1차 확장)
1. 범위
`내 파일` 섹션에서 파일과 폴더를 폴더로 드래그해 이동하는 기능을 적용한다.

2. 구현 레이어
`CloudRepository`-`ManageCloudUseCase`-`useCloudActions`-`cloud/page.tsx` 순으로 `moveFile`/`moveFolder` 흐름을 연결한다.

3. 상호작용 규칙
파일/폴더 드래그를 허용하고 드롭 타깃은 폴더로 제한한다.

4. 피드백 규칙
드롭 가능한 폴더에는 hover 강조를 적용하고 이동 중에는 중복 요청을 차단한다.

5. 예외 처리
같은 상위 경로로의 이동과 자기 자신 폴더로의 드롭은 무시하고 실패 시 콘솔 오류와 사용자 메시지로 원인을 알린다.

6. 검증
웹 타입체크/린트와 서버 테스트를 실행해 회귀를 확인한다.

## 10. Cloud 클립보드 액션 계획 (웹/데스크톱 2차)
1. 범위
Cloud 컨텍스트 메뉴에 복사/잘라내기/붙여넣기 액션을 추가하고 단축키를 연결한다.

2. 메뉴 위치
`more-vertical` 메뉴 상단에 복사/잘라내기/붙여넣기, 하단에 이름 변경/이동/다운로드/삭제를 배치한다.

3. 단축키
mac은 `cmd+x/c/v`, windows는 `ctrl+x/c/v`를 지원한다.

4. 가드 규칙
입력창/텍스트 에디터 포커스 상태에서는 단축키를 가로채지 않고 기본 동작을 유지한다.

5. 활성 범위
`내 파일`에서만 클립보드 액션을 활성화하고 최근/공유됨/중요/휴지통에서는 비활성 처리한다.

6. 복사 정책
파일 복사만 지원하고 폴더 복사는 미지원으로 고정한다.

## 11. Home 허브 현황 및 후속 확장
1. 현재 상태
Web은 `GET /api/v1/home/summary`를 사용해 오늘 일정, 지난/오늘 Todo, 최근 파일, 저장공간, 빠른 액션을 표시한다.

2. 구현 완료 범위
- 로그인 후 기본 진입을 `/home`으로 고정
- 사이드바 로고(`LB`) 클릭 시 Home 이동
- 빠른 액션 딥링크 연결
- 저장공간 카드와 파일 타입별 breakdown 표시

3. 후속 작업
- Desktop/Mobile 전용 Home 화면 확장
- 카드별 drill-down UX와 새로고침/에러 상태 정교화

4. 빠른 액션 연결
- `/calendar?quick=create`
- `/todo?quick=create`
- `/cloud?quick=upload`

5. 저장공간 시각화
저장공간 카드는 원형 도넛형 바를 사용하고 파일 타입별 사용량을 이미지/비디오/기타로 분해해 퍼센트와 용량을 함께 표시한다.

## 12. 캘린더 다중 계정 필터/색상 전략 현황
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

## 13. 다중 계정 특수 캘린더 + Todo 완료 동기화/출처 표기 통합 계획 (부분 폐기)
1. 유지 항목
Todo 완료 동기화, Todo 출처 표기, 완료 보존 정책은 유지한다.

2. 폐기 항목
특수 캘린더(휴일/생일) 선택/중복 병합 정책은 폐기하고, 공휴일 정책은 `15. KASI 공휴일 표시 현황`을 따른다.

## 14. Todo due 모델 및 정렬 기준 현황
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

6. due 없는 항목 처리
`due` 정렬에서 `due_date`가 없는 Todo는 항상 뒤로 보낸다.

## 15. Google Calendar API 에러 매핑 표준화 계획
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

## 16. KASI 공휴일 표시 현황
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
