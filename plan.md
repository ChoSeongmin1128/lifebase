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

## 8. 프론트 기능별 리팩토링 계획 (전 기능 공통 구조 전환)
1. 목표
웹/모바일의 구현된 모든 프론트 기능을 화면 중심 구현에서 기능 중심 구조로 전환한다.

2. 공통 패키지 구조
`packages/features/<feature>`에 `domain`, `usecase`, `repository`를 배치한다.

3. 앱별 어댑터 구조
웹/모바일은 각 앱 내부 `infrastructure`와 `ui/hooks`에서 공통 유스케이스를 연결한다.

4. 대상 기능
`auth`, `admin`, `cloud`, `gallery`, `calendar`, `todo`, `settings`를 전환 대상으로 한다.

5. 점진 마이그레이션 순서
1) 기능별 대표 흐름 선정
2) 응답 매핑 로직을 저장소 어댑터로 분리
3) 비즈니스 로직을 유스케이스로 추출
4) 유스케이스 테스트 보강
5) UI 페이지에서 훅으로 연결

6. 1차 구현 범위
Todo 생성 흐름을 웹/모바일 공통 패키지 기반으로 먼저 전환하고 나머지 기능은 동일 패턴으로 확장한다.

7. 확장 순서
2차 `calendar`, 3차 `cloud`, 4차 `settings`, 5차 `gallery`, 6차 `auth/admin` 순으로 전환한다.

8. 검증
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

## 11. Home 허브 1차 구축 계획 (웹 우선)
1. API 전략
`GET /api/v1/home/summary` 통합 엔드포인트로 오늘 요약 데이터를 단일 조회한다.

2. 요약 범위
오늘 일정, 지난/오늘 Todo, 최근 파일, 저장공간, 빠른 액션(일정/Todo/업로드)을 기본 카드로 제공한다.

3. 내비게이션 정책
로그인 후 기본 진입을 `/home`으로 고정하고 사이드바 로고(`LB`) 클릭 시 항상 Home으로 이동한다.

4. 빠른 액션 연결
Home 버튼은 인라인 생성 대신 딥링크를 사용한다.
- `/calendar?quick=create`
- `/todo?quick=create`
- `/cloud?quick=upload`

5. 저장공간 시각화
저장공간 카드는 원형 도넛형 바를 사용하고 파일 타입별 사용량을 이미지/비디오/기타로 분해해 퍼센트와 용량을 함께 표시한다.

## 12. 캘린더 다중 계정 필터/색상 전략 계획 (웹 1차)
1. 목표
Google Calendar 원본 색상 정합성과 다중 계정 가독성을 동시에 만족하는 색상 정책을 적용한다.

2. 색상 규칙
단일 계정 선택 시 Google 색상(`event.color_id -> calendar.color_id`)을 사용하고 다중 계정 동시 선택 시 계정 단위 통일 색상을 사용한다.

3. 계정 필터 규칙
캘린더 뷰 전체(Month/Week/3-Day/Agenda/Year)에 동일한 계정 필터를 적용한다.

4. 계정 선택 기본값
활성 Google 계정 전체 선택을 기본으로 하고 선택 상태는 사용자 설정으로 저장한다.

5. 백엔드 데이터 축
`calendars.google_account_id` 컬럼을 추가해 캘린더-계정 소속 관계를 명시적으로 관리한다.

6. 인증 API 확장
일반 사용자 대상 `GET /auth/google-accounts`, `POST /auth/google-accounts/link`를 추가해 다중 계정 조회/연결을 지원한다.

7. 웹 UX 확장
Settings > 일반에서 Google 계정 목록과 추가 연결 액션을 제공하고 캘린더 툴바에서 계정 멀티 선택을 지원한다.

8. 검증
서버 테스트와 웹 lint를 수행하고 단일 계정/다중 계정 색상 전환, 계정 필터 적용, 계정 추가 연결 시나리오를 수동 검증한다.

## 13. 다중 계정 특수 캘린더 + Todo 완료 동기화/출처 표기 통합 계획
1. Todo 완료 동기화
Google Tasks `hidden=true` 완료 항목 skip 로직을 제거하고 완료 항목을 정상 upsert한다.

2. Todo 출처 표기
`GET /todo/lists` 응답에 `google_account_email`을 포함해 UI에서 `Google · 계정이메일`로 표기한다.

3. 완료 보존 정책
`todo_done_retention_period` 설정(`1m|3m|6m|1y|3y|unlimited`)을 동기화 정리 cutoff 계산에 적용한다.

4. 특수 캘린더 선택
특수 캘린더는 `계정 선택 + 캘린더 선택` 이중 필터를 제공하고 설정 키로 저장한다.

5. 휴일 중복 정책
휴일은 표시 단계에서만 날짜+제목 기준으로 중복 병합하고 생일은 병합하지 않는다.

6. 호환 마이그레이션
`calendar_show_special_calendars=true` 기존 사용자 중 신규 선택값이 비어 있으면 1회 전체 선택으로 이관한다.

7. 검증
서버 테스트, 웹 빌드, 모바일 타입 검사를 수행해 Web/Mobile/Server 회귀를 확인한다.

## 14. Google Calendar API 에러 매핑 표준화 계획
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
