# CLAUDE.md

Lifebase용 에이전트 규칙 진입점.
이 파일은 짧게 유지하고, 세부 규칙은 분리 문서를 `@import`로 불러온다.

## 프로젝트 핵심
- 제품: 클라우드 + 캘린더 + Todo 통합
- 인증: Google OAuth only
- 플랫폼: Web, Desktop(macOS/Windows), Mobile(iOS/Android)
- 기획 인덱스: `@plan.md`

## 규칙 import
- 공통 워크플로우: `@.claude/rules/workflow.md`
- 커밋 메시지 규칙: `@.claude/rules/commit-message.md`
- 스킬 작성/운영 규칙: `@.claude/rules/skills.md`
- 보안/민감정보 규칙: `@.claude/rules/security.md`
- DB 컨벤션: `@.claude/rules/database.md`
- 프론트엔드 라우팅 컨벤션: `@.claude/rules/routing.md`
- API 아키텍처 규칙: `@.claude/rules/api-architecture.md`
- Google 연동 규칙: `@.claude/rules/google-api.md`
