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
/calendar/year/2026
/calendar/month/2026-02
/calendar/week/2026-W09
/calendar/day/2026-02-23

/cloud/folders/{folderId}
/cloud/files/{fileId}

/gallery?view=grid&sort=date

/tasks/{listId}

/settings/accounts
```
