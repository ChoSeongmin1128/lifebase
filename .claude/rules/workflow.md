# 워크플로우 규칙

## 목적
- 에이전트가 과도한 컨텍스트 없이 일관된 순서로 작업하도록 고정한다.
- 플랫폼 대응(Web/반응형 모바일 웹, Desktop macOS/Windows, Mobile iOS/Android)을 누락 없이 수행한다.

## 기본 순서
1. 탐색
- 요청과 직접 관련된 파일만 먼저 읽는다.
- 사실 확인이 필요한 항목은 코드/설정/로그로 검증한다.

2. 계획
- 변경 파일, 영향 범위, 검증 방법을 짧게 제시한다.
- 모호한 요구는 추정하지 말고 질문한다.

3. 구현
- 작은 단위로 수정한다.
- 요청 범위를 넘는 리팩터링/기능 추가를 하지 않는다.

4. 검증
- 가능한 테스트/검증 명령을 실행한다.
- 실행 불가 시 이유와 대체 검증 근거를 남긴다.

5. 보고
- 변경 사항, 이유, 리스크, 다음 액션을 간결히 보고한다.

## 플랫폼 대응 규칙
- 모든 기능 작업은 Web(+반응형 모바일 웹), Desktop(macOS/Windows), Mobile(iOS/Android) 대응 여부를 반드시 판단한다.
- 공통 범위가 있으면 Web에서 먼저 구현/검증한다.
- Web 공통 반영 후 Desktop/Mobile을 병렬로 작업하고 검증한다.

## 멀티 에이전트/워크트리 트리거
- 아래 조건 중 하나를 만족하면 멀티 에이전트 + git worktree 전략을 적용한다.
- 큰 단위 작업 2개 이상이 동시에 존재하고 수정 범위(파일/모듈)가 겹치는 경우
- 2개 이상 플랫폼을 동시에 터치하는 경우
- 큰 단위 작업의 정의: 독립 PR로 분리 가능한 기능/정책 단위
- 복합 작업/큰 단위 판정이 불명확하면 즉시 사용자에게 확인 질문을 한다.

## 멀티 에이전트 운영 원칙
- 기본 역할: explorer, worker, reviewer, monitor
- 병렬 한도: max_threads=6, max_depth=1
- explorer는 read-only 탐색, worker는 구현, reviewer는 리스크 검토, monitor는 장기 대기/폴링 담당

## 워크트리 운영 원칙
- 브랜치 네이밍: `task/<ticket>-<scope>-<platform>`
- 병합 전략: web-first 통합 브랜치
- 순서: Web 공통 반영 -> Desktop/Mobile rebase+merge -> 통합 검증

## 실행 정책 검증
- 레포 도메인 규칙은 `.claude/rules/*.md`로 운영한다.
- Codex 명령 실행 정책(강제)은 `.codex/rules/default.rules`로 운영한다.
- 표준 검증 명령: `codex execpolicy check --pretty --rules ./.codex/rules/default.rules -- <command>`
- 검증 결과가 기대한 `allow/prompt/forbidden`과 다르면 `.codex/rules/default.rules`를 수정하고 재검증한다.

## 용어 규칙
- LifeBase 내부 용어는 "Todo"로 통일한다 (할 일, 태스크 사용 금지)
- Google API를 지칭할 때만 "Google Tasks"를 사용한다
- 예: "Todo 생성", "Todo 완료", "Google Tasks 양방향 동기화"

## 컨텍스트 관리
- 컨텍스트가 길어지면 요약 후 진행한다.
- 주제가 바뀌면 불필요한 맥락을 끊고 새 작업에 집중한다.
