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
