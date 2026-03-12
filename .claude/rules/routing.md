# 프론트엔드 라우팅 컨벤션

## 핵심 원칙
- 화면에 보이는 상태는 URL에 반영한다
- 사용자가 새로고침하거나 URL을 공유했을 때 같은 화면이 나와야 한다

## URL 설계 규칙
- 페이지 전환이 있는 뷰는 경로(path)로 표현한다
- 같은 페이지 내 보기 모드/정렬 등 보조 상태는 쿼리 파라미터로 표현한다
- 모달/다이얼로그 등 임시 UI는 URL에 반영하지 않는다

## 예시
```
/calendar/year-compact/2026
/calendar/year-timeline/2026
/calendar/month/2026-02
/calendar/week/2026-W09
/calendar/3day/2026-02-23
/calendar/agenda

/cloud/folders/{folderId}
/cloud/files/{fileId}

/gallery?view=grid&sort=date

/todo/{listId}

/settings/general
/settings/calendar
/settings/todo
/settings/notifications
/settings/cloud
```

## 플랫폼 상태 복원 규칙
- Web/반응형 모바일 웹은 URL이 상태의 단일 소스다.
- 모바일 앱(iOS/Android)은 딥링크/네비게이션 파라미터로 동일 상태를 복원한다.
- Desktop 앱은 Web 경로 모델과 동일한 상태 의미를 유지한다.

## 라우트 전환 + Undo 규칙
- 라우트 전환이 함께 일어나는 destructive action(예: 현재 폴더 삭제)은 일반 목록 삭제와 같은 컴포넌트 로컬 pending/ref로 처리하지 않는다.
- `router.replace` 이후 컴포넌트가 다시 그려질 수 있으므로, Undo 세션과 optimistic hidden 상태는 라우트 전환 후에도 유지되는 범위에서 관리한다.
- 현재 폴더 삭제처럼 상위 폴더로 선이동하는 UX는 삭제 직후 부모 목록에서 항목이 즉시 숨겨져야 하며, 5초 후 실제 삭제까지 다시 보이면 안 된다.
- 같은 플로우의 Undo는 "원래 폴더로 강제 복귀"가 아니라, 명시 요구가 없으면 현재 머무는 상위 폴더에서 항목만 복구하는 것을 기본으로 한다.
- route change, refresh, cleanup effect가 Undo 전에 삭제를 자동 확정하지 못하게 하고, commit/undo 외의 숨은 finalize 경로를 만들지 않는다.

## 예외 규칙
- 보안/개인정보 노출 위험이 있는 상태는 URL에 직접 담지 않는다.
- 일회성 임시 상태(토스트 표시 여부 등)는 URL 동기화 대상에서 제외한다.

## 탭 순서
- Cloud / Calendar / Todo / Gallery / Settings (고정)
