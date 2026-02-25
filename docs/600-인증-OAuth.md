# 인증: OAuth + JWT

## 목적
- LifeBase의 인증/인가 구조를 정의한다.

## 인증 방식
- Google OAuth 2.0 단일 인증
- 한 번의 OAuth 인증으로 LifeBase의 모든 기능(클라우드, 캘린더, Todo, 갤러리, 동기화 등)을 사용할 수 있다
- 로컬 비밀번호 로그인은 지원하지 않는다

## 토큰 구조: OAuth + JWT
- Google OAuth로 사용자 인증 후, 서버에서 자체 JWT(Access Token + Refresh Token)를 발급한다
- 클라이언트는 이후 요청에 JWT Access Token을 사용한다
- Google OAuth 토큰은 서버에서 Google API 호출(Calendar/Tasks 동기화 등)에 사용한다

## LifeBase Refresh Token 정책
- 유효 기간: 30일
- 비밀번호 변경(Google 측) 감지 시 즉시 무효화
- 의심 활동(IP 급변 등) 시 강제 만료
- 사용자가 "모든 기기 로그아웃" 기능으로 전체 토큰 무효화 가능

## 토큰 갱신 흐름
1. 클라이언트가 API 요청 시 Access Token 만료 감지
2. 프론트엔드에서 백엔드 리프레시 API 호출
3. 서버가 Refresh Token 검증 후 새로운 Access Token 발급
4. 클라이언트가 새 토큰으로 원래 요청 재시도

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

## 사용자 격리 원칙
- 사용자 간 정보가 완전히 격리되어야 한다
- API 응답에 다른 사용자의 존재나 정보가 노출되지 않아야 한다

## 열린 질문
- 없음(모두 확정됨)
