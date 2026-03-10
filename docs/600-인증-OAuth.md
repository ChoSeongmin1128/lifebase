# 인증: OAuth + JWT

## 목적
- LifeBase의 인증/인가 구조를 정의한다.

## 인증 방식
- Google OAuth 2.0 단일 인증
- 한 번의 OAuth 인증으로 LifeBase의 모든 기능(클라우드, 캘린더, Todo, 갤러리, 동기화 등)을 사용할 수 있다
- 로컬 비밀번호 로그인은 지원하지 않는다

## 토큰 구조: OAuth + JWT
- Google OAuth로 사용자 인증 후, 서버에서 자체 JWT(Access Token + Refresh Token)를 발급한다
- Web/Admin은 이후 요청에 API가 심은 httpOnly 쿠키 세션을 사용하고, Mobile은 JWT Access Token 헤더를 사용한다
- Google OAuth 토큰은 서버에서 Google API 호출(Calendar/Tasks 동기화 등)에 사용한다
- LifeBase Access Token 유효 기간은 1시간이다. (`JWT_ACCESS_EXPIRY=1h`)
- LifeBase Refresh Token 유효 기간은 30일이다. (`JWT_REFRESH_EXPIRY=720h`)
- Refresh 시 기존 Refresh Token은 즉시 폐기하고, 새 Access Token + 새 Refresh Token을 다시 발급한다

## LifeBase Refresh Token 정책
- 유효 기간: 30일
- 비밀번호 변경(Google 측) 감지 시 즉시 무효화
- 의심 활동(IP 급변 등) 시 강제 만료
- 사용자가 "모든 기기 로그아웃" 기능으로 전체 토큰 무효화 가능

## 토큰 갱신 흐름
1. 클라이언트가 API 요청 시 Access Token 만료 감지
2. 프론트엔드에서 백엔드 리프레시 API 호출
3. 서버가 Refresh Token 검증 후 기존 Refresh Token을 폐기하고 새로운 Access Token + Refresh Token을 발급
4. Web/Admin은 새 세션 쿠키를 다시 심고, Mobile은 저장된 토큰 쌍을 새 값으로 교체
5. 클라이언트가 새 Access Token으로 원래 요청 재시도

## 클라이언트 갱신 정책
- Web은 Access Token 만료 임박 시 선제 갱신하고, 401 응답 시 1회 refresh 후 재시도한다. 민감 토큰은 브라우저 JS 저장소에 두지 않는다.
- Mobile도 401 응답 시 1회 refresh 후 재시도한다
- Refresh 실패 시 저장된 토큰을 삭제하고 로그인 화면으로 복귀한다

## Google OAuth Refresh Token 만료 시 처리
- Google 토큰이 만료되어도 LifeBase 자체 로그인은 유지한다
- 클라우드/Todo 등 LifeBase 자체 기능은 계속 사용 가능
- Google Calendar/Tasks 동기화만 일시 중단
- 처리 흐름:
  1. 서버가 Google API 호출 시 토큰 거부 감지
  2. 해당 사용자의 Google 연동 상태를 "재인증 필요"로 변경
  3. 클라이언트가 API 응답에서 `google_reauth_required` 플래그 수신
  4. 프론트엔드에서 배너/모달로 안내: "Google 계정 연결이 만료되었습니다. 다시 연결해주세요."
  5. 사용자가 Google OAuth 재인증 → 새 토큰 저장 → 동기화 복구

## 추가 Google 계정 관리
- 기본 계정: 로그인에 사용한 Google 계정이 곧 기본 동기화 계정
- 추가 계정: `user_google_accounts` 테이블에 별도 행으로 관리
- 각 계정별 독립된 토큰 라이프사이클(만료/재인증이 개별 동작)
- 추가 계정 연결 시 같은 OAuth 흐름을 타되, `login_hint` 파라미터로 다른 계정 선택 유도
- 추가 계정 해제 시 해당 토큰만 삭제, 동기화된 데이터 처리는 사용자에게 확인
- 데이터 구조:
  - `user_google_accounts`: id, user_id, google_email, access_token(암호화), refresh_token(암호화), token_expires_at, scopes, status(active/reauth_required/revoked), connected_at

## OAuth 스코프
- `openid` — 사용자 식별
- `email` — 이메일 기반 계정 매칭
- `profile` — 사용자 이름/프로필 사진
- `https://www.googleapis.com/auth/calendar` — Google Calendar 읽기/쓰기
- `https://www.googleapis.com/auth/tasks` — Google Tasks 읽기/쓰기
- Google Drive 스코프는 사용하지 않음 (자체 파일 스토리지 사용)

## 웹/관리자 OAuth 앱 구분
- 인증 URL 요청 시 `app` 파라미터를 사용한다.
  - 사용자 웹: `GET /api/v1/auth/url?app=web`
  - 관리자 웹: `GET /api/v1/auth/url?app=admin`
- 서버는 OAuth state에 앱 구분값을 포함해 서명하고, 콜백에서 검증한다.
- app별 redirect URI:
  - `web` → `${WEB_URL}/auth/callback`
  - `admin` → `${ADMIN_URL}/admin/auth/callback`
- Google OAuth 콘솔 Authorized redirect URI에 위 두 경로를 모두 등록해야 한다.
- `app=admin` 콜백은 Google 인증 성공만으로 완료되지 않는다. 서버는 기존 사용자와 `admin_users.is_active=true` 권한을 함께 확인한 뒤에만 admin 토큰을 발급한다.
- callback과 추가 Google 계정 연결(`POST /auth/google-accounts/link`)은 모두 서명된 `state`가 없으면 실패한다.

## 세션/비밀값 운영 규칙
- `JWT_SECRET`과 `STATE_HMAC_KEY`는 필수 운영 비밀값이다.
- 위 값이 비어 있거나 개발용 기본값이면 서버는 시작하지 않는다.
- Web/Admin 세션 쿠키는 `HttpOnly`, `SameSite=Lax`, `/api/v1` 경로 범위로 설정한다.

## 사용자 격리 원칙
- 사용자 간 정보가 완전히 격리되어야 한다
- API 응답에 다른 사용자의 존재나 정보가 노출되지 않아야 한다

## 열린 질문
- 없음(모두 확정됨)
