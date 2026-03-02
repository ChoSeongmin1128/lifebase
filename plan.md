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
