import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Toast } from "../components/Toast";
import { Button, Card, Input } from "../components/UI";
import { Server } from "lucide-react";
import { api, clearAdminToken, getAdminToken, setAdminToken } from "../lib/api";

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();

  const [adminKey, setAdminKey] = useState("");
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });

  useEffect(() => {
    const hasToken = Boolean((getAdminToken() || "").trim());
    if (hasToken) {
      // 验证 token 是否有效
      verifyToken();
    }
  }, [navigate]);

  const verifyToken = async () => {
    try {
      await api.getNodeConfig();
      navigate("/dashboard", { replace: true });
    } catch (e: any) {
      // token 无效，清除并重新登录
      if (e?.message?.includes("unauthorized") || e?.message?.includes("401")) {
        clearAdminToken();
      } else {
        // 其他错误，也清除 token
        clearAdminToken();
      }
    }
  };

  const canSubmit = useMemo(() => Boolean(!loading), [loading]);

  const submit = async () => {
    setLoading(true);
    try {
      // 调用登录接口，输入密码
      const res = await api.adminLogin(adminKey.trim());
      setAdminToken(res.token);
      setToast({ open: true, type: "success", message: "登录成功" });
      navigate("/dashboard", { replace: true });
    } catch (e: any) {
      clearAdminToken();
      setToast({ open: true, type: "error", message: e?.message || "登录失败" });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div
      style={{
        minHeight: "100vh",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        padding: 24,
        background:
          "radial-gradient(1200px 600px at 20% 0%, rgba(10,89,247,0.18), transparent 60%), radial-gradient(900px 500px at 100% 0%, rgba(65,186,65,0.10), transparent 55%), var(--bg-body)",
      }}
    >
      <div style={{ width: 560, maxWidth: "100%" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 14 }}>
          <div
            style={{
              width: 40,
              height: 40,
              background: "linear-gradient(135deg, #0A59F7 0%, #3275F9 100%)",
              borderRadius: 12,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              boxShadow: "0 10px 30px rgba(10, 89, 247, 0.22)",
            }}
          >
            <Server size={20} color="#fff" />
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
            <div style={{ fontSize: 18, fontWeight: 900 }}>Totoro Node 节点管理</div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>节点运维控制台</div>
          </div>
        </div>

        <Card glass title="登录">
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            <Input
              type="password"
              placeholder="请输入密码"
              value={adminKey}
              onChange={(e) => setAdminKey((e.target as any).value)}
              autoFocus
              onKeyDown={(e) => {
                if (e.key === "Enter") submit();
              }}
            />
            <div style={{ display: "flex", gap: 10, justifyContent: "flex-end" }}>
              <Button variant="primary" onClick={submit} disabled={!canSubmit}>
                {loading ? "登录中…" : "登录"}
              </Button>
            </div>
          </div>
        </Card>
      </div>

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};

