"use client";

import { useState, useEffect } from "react";

interface Document {
  id: string;
  name: string;
  data: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export default function Home() {
  const [apiUrl, setApiUrl] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [isConnected, setIsConnected] = useState(false);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedDoc, setSelectedDoc] = useState<Document | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isViewModalOpen, setIsViewModalOpen] = useState(false);
  const [docName, setDocName] = useState("");
  const [docData, setDocData] = useState("{}");
  const [editingId, setEditingId] = useState<string | null>(null);
  const [notification, setNotification] = useState<{ message: string; type: string } | null>(null);

  useEffect(() => {
    const savedUrl = localStorage.getItem("apiUrl");
    const savedKey = localStorage.getItem("apiKey");
    if (savedUrl && savedKey) {
      setApiUrl(savedUrl);
      setApiKey(savedKey);
      connectToApi(savedUrl, savedKey);
    }
  }, []);

  const showNotification = (message: string, type: string) => {
    setNotification({ message, type });
    setTimeout(() => setNotification(null), 3000);
  };

  const connectToApi = async (url: string, key: string) => {
    setLoading(true);
    try {
      const cleanUrl = url.replace(/\/$/, "");
      const res = await fetch(`${cleanUrl}/health`);
      if (!res.ok) throw new Error("Connection failed");

      localStorage.setItem("apiUrl", cleanUrl);
      localStorage.setItem("apiKey", key);
      setApiUrl(cleanUrl);
      setApiKey(key);
      setIsConnected(true);
      await loadDocuments(cleanUrl, key);
      showNotification("Connected successfully", "success");
    } catch {
      showNotification("Failed to connect", "error");
    } finally {
      setLoading(false);
    }
  };

  const loadDocuments = async (url?: string, key?: string) => {
    const targetUrl = url || apiUrl;
    const targetKey = key || apiKey;
    setLoading(true);
    try {
      const res = await fetch(`${targetUrl}/api/documents`, {
        headers: { "X-API-Key": targetKey },
      });
      const data = await res.json();
      setDocuments(data.data || []);
    } catch {
      showNotification("Failed to load documents", "error");
    } finally {
      setLoading(false);
    }
  };

  const handleConnect = (e: React.FormEvent) => {
    e.preventDefault();
    connectToApi(apiUrl, apiKey);
  };

  const disconnect = () => {
    localStorage.removeItem("apiUrl");
    localStorage.removeItem("apiKey");
    setIsConnected(false);
    setDocuments([]);
    setApiUrl("");
    setApiKey("");
  };

  const openCreateModal = () => {
    setEditingId(null);
    setDocName("");
    setDocData("{}");
    setIsModalOpen(true);
  };

  const openEditModal = (doc: Document) => {
    setEditingId(doc.id);
    setDocName(doc.name);
    setDocData(JSON.stringify(doc.data, null, 2));
    setIsModalOpen(true);
    setIsViewModalOpen(false);
  };

  const saveDocument = async (e: React.FormEvent) => {
    e.preventDefault();
    let parsedData;
    try {
      parsedData = JSON.parse(docData);
    } catch {
      showNotification("Invalid JSON", "error");
      return;
    }

    const method = editingId ? "PUT" : "POST";
    const url = editingId
      ? `${apiUrl}/api/documents/${editingId}`
      : `${apiUrl}/api/documents`;

    try {
      const res = await fetch(url, {
        method,
        headers: {
          "Content-Type": "application/json",
          "X-API-Key": apiKey,
        },
        body: JSON.stringify({ name: docName, data: parsedData }),
      });
      if (!res.ok) throw new Error();
      setIsModalOpen(false);
      loadDocuments();
      showNotification(editingId ? "Document updated" : "Document created", "success");
    } catch {
      showNotification("Failed to save", "error");
    }
  };

  const deleteDocument = async () => {
    if (!selectedDoc || !confirm(`Delete "${selectedDoc.name}"?`)) return;
    try {
      await fetch(`${apiUrl}/api/documents/${selectedDoc.id}`, {
        method: "DELETE",
        headers: { "X-API-Key": apiKey },
      });
      setIsViewModalOpen(false);
      loadDocuments();
      showNotification("Document deleted", "success");
    } catch {
      showNotification("Failed to delete", "error");
    }
  };

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text);
    showNotification(`${label} copied`, "success");
  };

  if (!isConnected) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 p-4">
        <div className="w-full max-w-md">
          <div className="text-center mb-8">
            <h1 className="text-2xl font-semibold text-gray-900 mb-2">JSON API</h1>
            <p className="text-gray-500">Connect to your API to manage documents</p>
          </div>
          <div className="bg-white border border-gray-200 rounded-lg p-6 shadow-sm">
            <form onSubmit={handleConnect} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">API URL</label>
                <input
                  type="url"
                  value={apiUrl}
                  onChange={(e) => setApiUrl(e.target.value)}
                  placeholder="https://your-api.onrender.com"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">API Key</label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder="Your API key"
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
                  required
                />
              </div>
              <button
                type="submit"
                disabled={loading}
                className="w-full bg-emerald-500 hover:bg-emerald-600 text-white font-medium py-2 px-4 rounded-md text-sm transition-colors disabled:opacity-50"
              >
                {loading ? "Connecting..." : "Connect"}
              </button>
            </form>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-white">
      {/* Notification */}
      {notification && (
        <div className={`fixed top-4 right-4 px-4 py-2 rounded-md text-sm z-50 ${notification.type === "success" ? "bg-emerald-50 text-emerald-700 border border-emerald-200" : "bg-red-50 text-red-700 border border-red-200"
          }`}>
          {notification.message}
        </div>
      )}

      {/* Header */}
      <header className="border-b border-gray-200">
        <div className="max-w-6xl mx-auto px-4 h-14 flex items-center justify-between">
          <h1 className="text-lg font-semibold text-gray-900">JSON API</h1>
          <div className="flex items-center gap-4">
            <span className="text-sm text-gray-500">{apiUrl}</span>
            <button onClick={disconnect} className="text-sm text-gray-500 hover:text-gray-700">
              Disconnect
            </button>
          </div>
        </div>
      </header>

      {/* Main */}
      <main className="max-w-6xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-xl font-semibold text-gray-900">Documents</h2>
            <p className="text-sm text-gray-500 mt-1">{documents.length} documents</p>
          </div>
          <button
            onClick={openCreateModal}
            className="bg-emerald-500 hover:bg-emerald-600 text-white font-medium py-2 px-4 rounded-md text-sm transition-colors"
          >
            New Document
          </button>
        </div>

        {loading ? (
          <div className="text-center py-12 text-gray-500">Loading...</div>
        ) : documents.length === 0 ? (
          <div className="text-center py-12 border border-gray-200 rounded-lg bg-gray-50">
            <p className="text-gray-500 mb-4">No documents yet</p>
            <button
              onClick={openCreateModal}
              className="text-emerald-600 hover:text-emerald-700 font-medium text-sm"
            >
              Create your first document
            </button>
          </div>
        ) : (
          <div className="border border-gray-200 rounded-lg overflow-hidden">
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-700">Name</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-700">ID</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-700">Updated</th>
                </tr>
              </thead>
              <tbody>
                {documents.map((doc) => (
                  <tr
                    key={doc.id}
                    onClick={() => { setSelectedDoc(doc); setIsViewModalOpen(true); }}
                    className="border-b border-gray-200 last:border-0 hover:bg-gray-50 cursor-pointer"
                  >
                    <td className="px-4 py-3 text-sm font-medium text-gray-900">{doc.name}</td>
                    <td className="px-4 py-3 text-sm text-gray-500 font-mono">{doc.id.slice(0, 8)}...</td>
                    <td className="px-4 py-3 text-sm text-gray-500">
                      {new Date(doc.updated_at).toLocaleDateString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* API Reference */}
        <div className="mt-12">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">API Reference</h3>
          <div className="grid gap-3">
            {[
              { method: "GET", path: "/api/documents", desc: "List all documents", auth: true },
              { method: "POST", path: "/api/documents", desc: "Create document", auth: true },
              { method: "GET", path: "/api/documents/:id", desc: "Get document", auth: true },
              { method: "PUT", path: "/api/documents/:id", desc: "Update document", auth: true },
              { method: "DELETE", path: "/api/documents/:id", desc: "Delete document", auth: true },
              { method: "GET", path: "/public/:id", desc: "Public read access", auth: false },
            ].map((ep, i) => (
              <div key={i} className="flex items-center gap-4 p-3 border border-gray-200 rounded-md">
                <span className={`text-xs font-mono font-medium px-2 py-1 rounded ${ep.method === "GET" ? "bg-emerald-50 text-emerald-700" :
                    ep.method === "POST" ? "bg-blue-50 text-blue-700" :
                      ep.method === "PUT" ? "bg-amber-50 text-amber-700" :
                        "bg-red-50 text-red-700"
                  }`}>{ep.method}</span>
                <code className="text-sm text-gray-700 flex-1">{ep.path}</code>
                <span className="text-sm text-gray-500">{ep.desc}</span>
                <span className="text-xs text-gray-400">{ep.auth ? "Auth required" : "Public"}</span>
              </div>
            ))}
          </div>
        </div>
      </main>

      {/* Create/Edit Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg w-full max-w-lg shadow-xl">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
              <h3 className="text-lg font-semibold">{editingId ? "Edit Document" : "New Document"}</h3>
              <button onClick={() => setIsModalOpen(false)} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
            </div>
            <form onSubmit={saveDocument} className="p-6 space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input
                  type="text"
                  value={docName}
                  onChange={(e) => setDocName(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">JSON Data</label>
                <textarea
                  value={docData}
                  onChange={(e) => setDocData(e.target.value)}
                  rows={10}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm font-mono"
                />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setIsModalOpen(false)} className="px-4 py-2 text-sm text-gray-700 hover:text-gray-900">Cancel</button>
                <button type="submit" className="bg-emerald-500 hover:bg-emerald-600 text-white font-medium py-2 px-4 rounded-md text-sm">Save</button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* View Modal */}
      {isViewModalOpen && selectedDoc && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg w-full max-w-2xl shadow-xl max-h-[90vh] overflow-auto">
            <div className="flex items-center justify-between px-6 py-4 border-b border-gray-200">
              <h3 className="text-lg font-semibold">{selectedDoc.name}</h3>
              <button onClick={() => setIsViewModalOpen(false)} className="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
            </div>
            <div className="p-6 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-gray-500 mb-1">ID</label>
                  <div className="flex gap-2">
                    <code className="text-sm text-gray-700 bg-gray-50 px-2 py-1 rounded flex-1 truncate">{selectedDoc.id}</code>
                    <button onClick={() => copyToClipboard(selectedDoc.id, "ID")} className="text-xs text-gray-500 hover:text-gray-700">Copy</button>
                  </div>
                </div>
                <div>
                  <label className="block text-xs text-gray-500 mb-1">Updated</label>
                  <p className="text-sm text-gray-700">{new Date(selectedDoc.updated_at).toLocaleString()}</p>
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Public URL</label>
                <div className="flex gap-2">
                  <code className="text-sm text-gray-700 bg-gray-50 px-2 py-1 rounded flex-1 truncate">{apiUrl}/public/{selectedDoc.id}</code>
                  <button onClick={() => copyToClipboard(`${apiUrl}/public/${selectedDoc.id}`, "URL")} className="text-xs text-gray-500 hover:text-gray-700">Copy</button>
                </div>
              </div>
              <div>
                <label className="block text-xs text-gray-500 mb-1">Data</label>
                <pre className="text-sm bg-gray-50 border border-gray-200 rounded-md p-4 overflow-auto max-h-64">
                  {JSON.stringify(selectedDoc.data, null, 2)}
                </pre>
              </div>
              <div className="flex justify-between pt-2">
                <button onClick={deleteDocument} className="text-red-600 hover:text-red-700 text-sm">Delete</button>
                <button onClick={() => openEditModal(selectedDoc)} className="bg-emerald-500 hover:bg-emerald-600 text-white font-medium py-2 px-4 rounded-md text-sm">Edit</button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
