import React, { useEffect, useMemo, useState } from "react";
import { Plus, RefreshCw, Trash2, Upload } from "lucide-react";
import { Toast } from "../components/Toast";
import { Button, Card, Input, Pagination, SearchInput, Select } from "../components/UI";
import { api, DeviceWhitelistRow } from "../lib/api";

type EditState = {
  open: boolean;
  mode: "create" | "edit";
  row: Partial<DeviceWhitelistRow>;
};

export const DeviceWhitelistPage: React.FC = () => {
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });
  const [loading, setLoading] = useState(false);

  const [q, setQ] = useState("");
  const [enabledFilter, setEnabledFilter] = useState<"all" | "enabled" | "disabled">("all");

  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);

  const [rows, setRows] = useState<DeviceWhitelistRow[]>([]);
  const [total, setTotal] = useState(0);
  const [toggling, setToggling] = useState<Record<string, boolean>>({});

  const [edit, setEdit] = useState<EditState>({ open: false, mode: "create", row: { enabled: true } as any });
  const [saving, setSaving] = useState(false);

  const [importOpen, setImportOpen] = useState(false);
  const [importText, setImportText] = useState("");
  const [importing, setImporting] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [dragOver, setDragOver] = useState(false);
  const fileInputId = "whitelist-import-file";

  const offset = (page - 1) * pageSize;

  const refresh = async () => {
    setLoading(true);
    try {
      const data = await api.whitelistList({ limit: pageSize, offset });
      setRows(data.devices || []);
      setTotal(data.total || 0);
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "加载失败" });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize]);

  const filtered = useMemo(() => {
    const qq = q.trim().toLowerCase();
    return (rows || []).filter((r) => {
      if (enabledFilter === "enabled" && !r.enabled) return false;
      if (enabledFilter === "disabled" && r.enabled) return false;
      if (!qq) return true;
      const hay = [r.device_id, r.mac, r.note].join(" ").toLowerCase();
      return hay.includes(qq);
    });
  }, [rows, q, enabledFilter]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const openCreate = () => setEdit({ open: true, mode: "create", row: { enabled: true } as any });
  const openEdit = (r: DeviceWhitelistRow) => setEdit({ open: true, mode: "edit", row: { ...r } });

  const canSave = useMemo(() => {
    const r = edit.row || {};
    return Boolean(String(r.device_id || "").trim() && !saving);
  }, [edit.row, saving]);

  const save = async () => {
    const r = edit.row || {};
    const req = {
      device_id: String(r.device_id || "").trim(),
      mac: String(r.mac || "").trim(),
      enabled: Boolean((r as any).enabled),
      note: String(r.note || "").trim(),
    };
    if (!req.device_id) {
      setToast({ open: true, type: "error", message: "device_id 为必填" });
      return;
    }
    setSaving(true);
    try {
      await api.whitelistUpsert(req);
      setToast({ open: true, type: "success", message: "保存成功" });
      setEdit((s) => ({ ...s, open: false }));
      await refresh();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "保存失败" });
    } finally {
      setSaving(false);
    }
  };

  const del = async (deviceID: string) => {
    const ok = window.confirm(`确认删除白名单设备：${deviceID} ？`);
    if (!ok) return;
    try {
      await api.whitelistDelete({ device_id: deviceID });
      setToast({ open: true, type: "success", message: "已删除" });
      await refresh();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "删除失败" });
    }
  };

  const toggleEnabled = async (r: DeviceWhitelistRow) => {
    const deviceID = String(r.device_id || "").trim();
    if (!deviceID) return;
    if (toggling[deviceID]) return;
    const nextEnabled = !Boolean(r.enabled);
    setToggling((m) => ({ ...m, [deviceID]: true }));
    // 乐观更新
    setRows((prev) => prev.map((x) => (x.device_id === deviceID ? { ...x, enabled: nextEnabled } : x)));
    try {
      await api.whitelistUpsert({
        device_id: deviceID,
        mac: String(r.mac || "").trim(),
        enabled: nextEnabled,
        note: String(r.note || "").trim(),
      });
      setToast({ open: true, type: "success", message: nextEnabled ? "已启用" : "已停用" });
    } catch (e: any) {
      // 回滚
      setRows((prev) => prev.map((x) => (x.device_id === deviceID ? { ...x, enabled: !nextEnabled } : x)));
      setToast({ open: true, type: "error", message: e?.message || "操作失败" });
    } finally {
      setToggling((m) => ({ ...m, [deviceID]: false }));
    }
  };

  const doImport = async () => {
    const csv = importText.trim();
    if (!csv) {
      setToast({ open: true, type: "error", message: "导入内容为空" });
      return;
    }
    setImporting(true);
    try {
      const res = await api.whitelistImport({ csv });
      setToast({ open: true, type: "success", message: `导入完成：成功 ${res.imported}，跳过 ${res.skipped}` });
      setImportOpen(false);
      setImportText("");
      await refresh();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "导入失败" });
    } finally {
      setImporting(false);
    }
  };

  const doImportFile = async () => {
    if (!importFile) {
      setToast({ open: true, type: "error", message: "请选择要导入的表格文件" });
      return;
    }
    setImporting(true);
    try {
      const res = await api.whitelistImportFile(importFile);
      setToast({ open: true, type: "success", message: `导入完成：成功 ${res.imported}，跳过 ${res.skipped}` });
      setImportOpen(false);
      setImportText("");
      setImportFile(null);
      await refresh();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "导入失败" });
    } finally {
      setImporting(false);
    }
  };

  return (
    <div>
      <Card
        title="设备白名单"
        extra={
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <Button variant="outline" onClick={refresh} disabled={loading} style={{ height: 40 }}>
              <RefreshCw size={16} /> 刷新
            </Button>
            <Button variant="outline" onClick={() => setImportOpen(true)} style={{ height: 40 }}>
              <Upload size={16} /> 批量导入
            </Button>
            <Button variant="primary" onClick={openCreate} style={{ height: 40 }}>
              <Plus size={16} /> 新增
            </Button>
          </div>
        }
      >
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center", marginBottom: 14 }}>
          <SearchInput placeholder="搜索 设备ID / MAC / 备注…" value={q} onChange={(e) => setQ((e.target as any).value)} />
          <Select
            value={enabledFilter}
            onChange={(v) => setEnabledFilter(v as any)}
            options={[
              { label: "全部", value: "all" },
              { label: "启用", value: "enabled" },
              { label: "禁用", value: "disabled" },
            ]}
          />
          <Select
            value={String(pageSize)}
            onChange={(v) => {
              setPage(1);
              setPageSize(Number(v));
            }}
            options={[
              { label: "50/页", value: "50" },
              { label: "100/页", value: "100" },
              { label: "200/页", value: "200" },
            ]}
          />
          <div style={{ marginLeft: "auto", fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>
            {loading ? "加载中…" : `本页 ${filtered.length} 条 / 总计 ${total} 条`}
          </div>
        </div>

        <table className="table">
          <thead>
            <tr>
              <th>设备ID</th>
              <th>MAC</th>
              <th>状态</th>
              <th>备注</th>
              <th>更新时间</th>
              <th style={{ width: 140 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((r) => (
              <tr key={r.device_id}>
                <td style={{ fontWeight: 900 }}>{r.device_id}</td>
                <td style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace" }}>
                  {r.mac || "-"}
                </td>
                <td>
                  <button
                    className="btn btn-outline"
                    style={{
                      height: 34,
                      padding: "0 12px",
                      fontWeight: 900,
                      color: r.enabled ? "var(--success)" : "var(--error)",
                      opacity: toggling[r.device_id] ? 0.7 : 1,
                    }}
                    disabled={Boolean(toggling[r.device_id])}
                    onClick={() => toggleEnabled(r)}
                    title={r.enabled ? "点击停用" : "点击启用"}
                  >
                    {toggling[r.device_id] ? "处理中…" : r.enabled ? "已启用" : "已停用"}
                  </button>
                </td>
                <td style={{ color: "var(--text-secondary)", fontWeight: 800 }}>{r.note || "-"}</td>
                <td style={{ color: "var(--text-secondary)", fontWeight: 800 }}>{r.updated_at || "-"}</td>
                <td>
                  <div style={{ display: "flex", gap: 10 }}>
                    <button className="btn btn-outline" style={{ height: 36, padding: "0 12px" }} onClick={() => openEdit(r)}>
                      编辑
                    </button>
                    <button
                      className="btn btn-outline"
                      style={{ height: 36, padding: "0 12px", color: "var(--error)" }}
                      onClick={() => del(r.device_id)}
                      title="删除"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {!filtered.length && (
              <tr>
                <td colSpan={6} style={{ color: "var(--text-secondary)", fontWeight: 800 }}>
                  暂无数据
                </td>
              </tr>
            )}
          </tbody>
        </table>

        <Pagination currentPage={page} totalPages={totalPages} onPageChange={setPage} />
      </Card>

      {edit.open && (
        <div className="modal-overlay" onMouseDown={() => setEdit((s) => ({ ...s, open: false }))}>
          <div className="card glass modal-panel" onMouseDown={(e) => e.stopPropagation()} style={{ padding: 22 }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
              <div style={{ fontSize: 18, fontWeight: 900 }}>{edit.mode === "create" ? "新增白名单设备" : "编辑白名单设备"}</div>
              <button className="btn btn-ghost" style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }} onClick={() => setEdit((s) => ({ ...s, open: false }))}>
                ×
              </button>
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 12 }}>
              <div style={{ gridColumn: "1 / -1" }}>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>设备ID *</div>
                <Input
                  value={String(edit.row.device_id || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, row: { ...s.row, device_id: (e.target as any).value } }))}
                  placeholder="例如 DEV001"
                  disabled={edit.mode === "edit"}
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>MAC 地址</div>
                <Input
                  value={String(edit.row.mac || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, row: { ...s.row, mac: (e.target as any).value } }))}
                  placeholder="可选，例如 a0:78:17:a1:a9:4a"
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>启用状态</div>
                <label style={{ display: "flex", alignItems: "center", gap: 8, fontWeight: 900, fontSize: 13, height: 44 }}>
                  <input
                    type="checkbox"
                    checked={Boolean((edit.row as any).enabled)}
                    onChange={(e) => setEdit((s) => ({ ...s, row: { ...s.row, enabled: e.target.checked } as any }))}
                  />
                  启用该设备
                </label>
              </div>
              <div style={{ gridColumn: "1 / -1" }}>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>备注</div>
                <Input value={String(edit.row.note || "")} onChange={(e) => setEdit((s) => ({ ...s, row: { ...s.row, note: (e.target as any).value } }))} placeholder="可选：用于标注用途/归属" />
              </div>
            </div>

            <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 16 }}>
              <Button variant="outline" onClick={() => setEdit((s) => ({ ...s, open: false }))} disabled={saving}>
                取消
              </Button>
              <Button variant="primary" onClick={save} disabled={!canSave}>
                {saving ? "保存中…" : "保存"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {importOpen && (
        <div className="modal-overlay" onMouseDown={() => setImportOpen(false)}>
          <div className="card glass modal-panel" onMouseDown={(e) => e.stopPropagation()} style={{ padding: 22 }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
              <div style={{ fontSize: 18, fontWeight: 900 }}>批量导入</div>
              <button className="btn btn-ghost" style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }} onClick={() => setImportOpen(false)}>
                ×
              </button>
            </div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 10 }}>
              支持上传 Excel 表格（.xlsx/.xls），或粘贴文本导入。
            </div>

            <div className="glass" style={{ padding: 14, borderRadius: 16, marginBottom: 12 }}>
              <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 8 }}>上传表格</div>
              {/* 隐藏原生 input，通过自定义 UI 触发 */}
              <input
                id={fileInputId}
                type="file"
                accept=".xlsx,.xls"
                style={{ display: "none" }}
                onChange={(e) => {
                  const f = (e.target as any).files?.[0] || null;
                  setImportFile(f);
                }}
              />

              <div
                className="glass"
                style={{
                  padding: 14,
                  borderRadius: 14,
                  border: dragOver ? "1px solid rgba(10, 89, 247, 0.65)" : "1px dashed rgba(0,0,0,0.10)",
                  background: dragOver ? "rgba(10, 89, 247, 0.06)" : "rgba(255,255,255,0.55)",
                  transition: "all .15s ease",
                }}
                onDragEnter={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setDragOver(true);
                }}
                onDragOver={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setDragOver(true);
                }}
                onDragLeave={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setDragOver(false);
                }}
                onDrop={(e) => {
                  e.preventDefault();
                  e.stopPropagation();
                  setDragOver(false);
                  const f = (e.dataTransfer as any)?.files?.[0] || null;
                  if (!f) return;
                  const name = String(f.name || "").toLowerCase();
                  if (!name.endsWith(".xlsx") && !name.endsWith(".xls")) {
                    setToast({ open: true, type: "error", message: "仅支持 .xlsx/.xls 文件" });
                    return;
                  }
                  setImportFile(f);
                }}
              >
                <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
                  <div style={{ display: "flex", flexDirection: "column", gap: 4, minWidth: 0 }}>
                    <div style={{ fontSize: 13, fontWeight: 900 }}>
                      {importFile ? "已选择文件" : dragOver ? "松开以选择文件" : "拖拽文件到此处，或点击选择"}
                    </div>
                    <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                      {importFile ? importFile.name : "支持 .xlsx / .xls"}
                    </div>
                  </div>
                  <label htmlFor={fileInputId}>
                    <button className="btn btn-outline" style={{ height: 36, padding: "0 12px" }} disabled={importing} type="button">
                      <Upload size={16} />
                      选择文件
                    </button>
                  </label>
                </div>

                {importFile && (
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 10, marginTop: 10 }}>
                    <button
                      className="btn btn-ghost"
                      style={{ height: 32, padding: "0 10px", color: "var(--text-secondary)" }}
                      type="button"
                      onClick={() => {
                        setImportFile(null);
                        const el = document.getElementById(fileInputId) as any;
                        if (el) el.value = "";
                      }}
                      disabled={importing}
                      title="移除已选文件"
                    >
                      移除
                    </button>
                    <Button variant="primary" onClick={doImportFile} disabled={importing || !importFile} style={{ height: 36 }}>
                      {importing ? "导入中…" : "上传并导入"}
                    </Button>
                  </div>
                )}
              </div>
            </div>

            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 8 }}>文本导入</div>
            <textarea
              className="input"
              value={importText}
              onChange={(e) => setImportText((e.target as any).value)}
              placeholder={`示例：\nDEV001,1,生产一号\nDEV002,0,临时禁用`}
              style={{ minHeight: 160, borderRadius: 16, resize: "vertical" }}
            />
            <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 14 }}>
              <Button variant="outline" onClick={() => setImportOpen(false)} disabled={importing}>
                取消
              </Button>
              <Button variant="primary" onClick={doImport} disabled={importing || !importText.trim()}>
                {importing ? "导入中…" : "开始导入"}
              </Button>
            </div>
          </div>
        </div>
      )}

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};


