# API 아키텍처 규칙

## 계층 규칙
- 1 API Endpoint = 1 Controller = 1 UseCase
- Controller는 입력 검증/응답 매핑만 담당한다
- 비즈니스 로직은 UseCase에만 둔다
- 도메인 규칙은 domain 레이어에 둔다

## 헥사고날 디렉토리 구조
```
internal/<module>/domain         — 엔티티, 값 객체, 도메인 규칙
internal/<module>/usecase        — 유스케이스(애플리케이션 서비스)
internal/<module>/port/in        — 입력 포트(인터페이스)
internal/<module>/port/out       — 출력 포트(저장소/외부 API 인터페이스)
internal/<module>/adapter/in/http — 컨트롤러, DTO, 라우트
internal/<module>/adapter/out/*   — DB, 파일시스템, Google API 어댑터
internal/shared                  — 공통 에러, 트랜잭션 경계, 로깅, 관측성
```

## API 설계 규칙
- Prefix: `/api/v1/`
- 페이지네이션: Cursor 기반 (`?cursor=abc&limit=50`)
- 에러 포맷: `{ "error": { "code": "FILE_NOT_FOUND", "message": "..." } }`
- Rate limiting: 클라이언트당 100 req/min
- Health check: `GET /api/v1/health`
- CORS: `api.lifebase.cc`에서 `web.lifebase.cc` origin 허용
- WebSocket 재연결: 지수 백오프 (1s → 2s → 4s → ... 최대 30s)

## 서버 중심 원칙
- 권한, 용량, 공유, 동기화 충돌 정책은 서버가 최종 판단한다
- 기능 규칙은 서버에서 단일 관리한다 (플랫폼별 규칙 분기 금지)
