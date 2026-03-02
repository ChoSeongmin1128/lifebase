"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getAdminAccessToken } from "@/lib/admin-auth";
import { adminApi } from "@/lib/admin-api";
import { formatBytes, splitBytes, toBytes, type ByteUnit } from "@/lib/bytes";

type GoogleAccount = {
  ID: string;
  GoogleEmail: string;
  GoogleID: string;
  Status: "active" | "reauth_required" | "revoked";
  IsPrimary: boolean;
  ConnectedAt: string;
};

type UserDetail = {
  ID: string;
  Email: string;
  Name: string;
  StorageQuotaBytes: number;
  StorageUsedBytes: number;
};

export default function AdminUserDetailPage() {
  const params = useParams<{ userId: string }>();
  const userID = params.userId;

  const [user, setUser] = useState<UserDetail | null>(null);
  const [accounts, setAccounts] = useState<GoogleAccount[]>([]);
  const [quotaValue, setQuotaValue] = useState("");
  const [quotaUnit, setQuotaUnit] = useState<ByteUnit>("GB");
  const [confirm, setConfirm] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const statusLabel: Record<GoogleAccount["Status"], string> = {
    active: "정상",
    reauth_required: "재인증 필요",
    revoked: "해지됨",
  };

  const expectedConfirm = useMemo(() => (user ? `DELETE ${user.Email}` : ""), [user]);

  const load = useCallback(async () => {
    const token = getAdminAccessToken();
    if (!token) return;
    setLoading(true);
    setError(null);
    setMessage(null);
    try {
      const data = await adminApi<{ user: UserDetail; google_accounts: GoogleAccount[] }>(`/admin/users/${userID}`, {
        token,
      });
      setUser(data.user);
      setAccounts(data.google_accounts);
      const quota = splitBytes(data.user.StorageQuotaBytes);
      setQuotaValue(quota.value);
      setQuotaUnit(quota.unit);
    } catch (e) {
      setError(e instanceof Error ? e.message : "조회 실패");
    } finally {
      setLoading(false);
    }
  }, [userID]);

  useEffect(() => {
    load();
  }, [load]);

  const saveQuota = async () => {
    const token = getAdminAccessToken();
    if (!token || !user) return;
    setError(null);
    setMessage(null);
    try {
      const quotaBytes = toBytes(quotaValue, quotaUnit);
      if (quotaBytes === null) {
        setError("할당량은 0보다 큰 숫자로 입력해 주세요.");
        return;
      }

      await adminApi(`/admin/users/${user.ID}/quota`, {
        method: "PATCH",
        token,
        body: { quota_bytes: quotaBytes },
      });
      setMessage("할당량을 업데이트했습니다.");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "할당량 업데이트 실패");
    }
  };

  const recalc = async () => {
    const token = getAdminAccessToken();
    if (!token || !user) return;
    setError(null);
    setMessage(null);
    try {
      await adminApi(`/admin/users/${user.ID}/recalculate-storage`, {
        method: "POST",
        token,
      });
      setMessage("사용량 재계산을 완료했습니다.");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "재계산 실패");
    }
  };

  const resetStorage = async () => {
    const token = getAdminAccessToken();
    if (!token || !user) return;
    setError(null);
    setMessage(null);
    try {
      await adminApi(`/admin/users/${user.ID}/reset-storage`, {
        method: "POST",
        token,
        body: { confirm },
      });
      setMessage("스토리지 초기화를 완료했습니다.");
      setConfirm("");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "스토리지 초기화 실패");
    }
  };

  const changeAccountStatus = async (accountID: string, status: GoogleAccount["Status"]) => {
    const token = getAdminAccessToken();
    if (!token || !user) return;
    setError(null);
    setMessage(null);
    try {
      await adminApi(`/admin/users/${user.ID}/google-accounts/${accountID}/status`, {
        method: "PATCH",
        token,
        body: { status },
      });
      setMessage("Google 계정 상태를 변경했습니다.");
      load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "상태 변경 실패");
    }
  };

  if (loading && !user) return <p className="text-sm text-text-muted">로딩 중...</p>;
  if (!user) return <p className="text-sm text-error">{error || "사용자를 찾을 수 없습니다."}</p>;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-text-strong">사용자 상세 운영</h1>
        <p className="mt-1 text-sm text-text-secondary">
          {user.Email} ({user.ID})
        </p>
      </div>

      {message ? <p className="text-sm text-success">{message}</p> : null}
      {error ? <p className="text-sm text-error">{error}</p> : null}

      <section className="rounded-lg border border-border bg-surface p-4">
        <h2 className="text-sm font-semibold text-text-strong">저장공간</h2>
        <p className="mt-2 text-sm text-text-secondary">
          사용량: <span className="tabular-nums">{formatBytes(user.StorageUsedBytes)}</span> / 할당량:{" "}
          <span className="tabular-nums">{formatBytes(user.StorageQuotaBytes)}</span>
        </p>

        <div className="mt-3 flex flex-wrap items-center gap-2">
          <Input
            type="number"
            min="0.01"
            step="0.01"
            value={quotaValue}
            onChange={(e) => setQuotaValue(e.target.value)}
            className="w-32"
          />
          <select
            className="h-9 rounded-lg border border-border bg-surface px-2 text-sm"
            value={quotaUnit}
            onChange={(e) => setQuotaUnit(e.target.value as ByteUnit)}
          >
            <option value="B">B</option>
            <option value="KB">KB</option>
            <option value="MB">MB</option>
            <option value="GB">GB</option>
            <option value="TB">TB</option>
          </select>
          <Button variant="secondary" onClick={saveQuota}>
            할당량 저장
          </Button>
          <Button variant="secondary" onClick={recalc}>
            사용량 재계산
          </Button>
        </div>
        <p className="mt-1 text-xs text-text-muted">숫자와 단위를 나눠 입력한 뒤 저장하세요.</p>

        <div className="mt-4 rounded-lg border border-error/30 bg-error/5 p-3">
          <p className="text-sm font-medium text-error">위험 작업: 스토리지 초기화</p>
          <p className="mt-1 text-xs text-text-secondary">
            실제 파일과 DB 메타데이터를 삭제합니다. 이 사용자가 소유한 공유 폴더가 있으면 다른 사용자도 접근할 수 없습니다.
          </p>
          <p className="mt-1 text-xs text-text-secondary">
            확인 문구: <span className="font-mono text-text-strong">{expectedConfirm}</span>
          </p>
          <div className="mt-2 flex items-center gap-2">
            <Input value={confirm} onChange={(e) => setConfirm(e.target.value)} placeholder={expectedConfirm} />
            <Button variant="danger" disabled={confirm !== expectedConfirm} onClick={resetStorage}>
              스토리지 초기화
            </Button>
          </div>
        </div>
      </section>

      <section className="rounded-lg border border-border bg-surface p-4">
        <h2 className="text-sm font-semibold text-text-strong">Google 계정 상태</h2>
        <div className="mt-2 rounded-lg border border-border bg-surface-accent/40 p-3 text-xs text-text-secondary">
          <p className="font-medium text-text-primary">운영 안내</p>
          <p className="mt-1">아래 상태 변경은 사용자 Google 연동 동기화 동작에 직접 영향을 줍니다.</p>
          <p className="mt-1"><span className="font-medium text-text-primary">정상</span>: 동기화 허용 상태</p>
          <p className="mt-1"><span className="font-medium text-text-primary">재인증 필요</span>: 사용자 재로그인/권한 재동의 필요 상태</p>
          <p className="mt-1"><span className="font-medium text-text-primary">해지</span>: 연동 차단 상태(동기화 중단)</p>
        </div>
        <div className="mt-3 space-y-2">
          {accounts.map((account) => (
            <div key={account.ID} className="rounded border border-border p-3">
              <div className="text-sm text-text-strong">
                {account.GoogleEmail} {account.IsPrimary ? "(주 계정)" : ""}
              </div>
              <div className="mt-1 text-xs text-text-secondary">상태: {statusLabel[account.Status]}</div>
              <div className="mt-2 flex flex-wrap gap-2">
                <Button variant="secondary" size="sm" onClick={() => changeAccountStatus(account.ID, "active")}>
                  정상
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => changeAccountStatus(account.ID, "reauth_required")}
                >
                  재인증 필요
                </Button>
                <Button variant="secondary" size="sm" onClick={() => changeAccountStatus(account.ID, "revoked")}>
                  해지
                </Button>
              </div>
            </div>
          ))}
          {accounts.length === 0 ? <p className="text-sm text-text-muted">연결된 Google 계정이 없습니다.</p> : null}
        </div>
      </section>
    </div>
  );
}
