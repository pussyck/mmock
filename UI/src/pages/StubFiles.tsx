import { useEffect, useMemo, useState } from "react";
import { FileText, Folder, RefreshCw, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import Editor from "@monaco-editor/react";
import { stubApi, StubContent, StubFile } from "@/lib/api";
import { toast } from "sonner";

interface GroupedFiles {
  folder: string;
  files: StubFile[];
}

export default function StubFiles() {
  const [files, setFiles] = useState<StubFile[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState("");
  const [selectedFile, setSelectedFile] = useState<StubFile | null>(null);
  const [fileContent, setFileContent] = useState<StubContent | null>(null);
  const [contentLoading, setContentLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [openFolders, setOpenFolders] = useState<Record<string, boolean>>({});
  const [editedContent, setEditedContent] = useState<string>("");

  const editorLanguage = useMemo(() => {
    if (!selectedFile) return "plaintext";
    const lower = selectedFile.path.toLowerCase();
    if (lower.endsWith(".json")) return "json";
    if (lower.endsWith(".yaml") || lower.endsWith(".yml")) return "yaml";
    return "plaintext";
  }, [selectedFile]);

  const fetchFiles = async () => {
    setLoading(true);
    try {
      const data = await stubApi.getAll();
      setFiles(data || []);
    } catch (error) {
      toast.error("Failed to fetch stub files");
      console.error(error);
    } finally {
      setLoading(false);
    }
  };

  const fetchContent = async (file: StubFile) => {
    setContentLoading(true);
    setSelectedFile(file);
    try {
      const content = await stubApi.getContent(file.path);
      setFileContent(content);
      setEditedContent(content.content);
    } catch (error) {
      toast.error("Failed to load file content");
      console.error(error);
      setFileContent(null);
      setEditedContent("");
    } finally {
      setContentLoading(false);
    }
  };

  const handleSave = async () => {
    if (!selectedFile) return;
    setSaving(true);
    try {
      const updated = await stubApi.updateContent(selectedFile.path, editedContent);
      setFileContent(updated);
      // Refresh file list (sizes, new files, etc.)
      await fetchFiles();
      // Keep header size in sync with actual content length
      setSelectedFile({
        ...selectedFile,
        size: updated.content.length,
      });
      toast.success("File saved");
    } catch (error) {
      toast.error("Failed to save file");
      console.error(error);
    } finally {
      setSaving(false);
    }
  };

  useEffect(() => {
    fetchFiles();
  }, []);

  const filteredFiles = useMemo(() => {
    if (!searchQuery) return files;
    const q = searchQuery.toLowerCase();
    return files.filter((f) => f.path.toLowerCase().includes(q));
  }, [files, searchQuery]);

  const groupedFiles: GroupedFiles[] = useMemo(() => {
    const groups: Record<string, StubFile[]> = {};

    for (const file of filteredFiles) {
      const lastSlash = file.path.lastIndexOf("/");
      const folder = lastSlash > 0 ? file.path.slice(0, lastSlash) : "/";

      if (!groups[folder]) {
        groups[folder] = [];
      }
      groups[folder].push(file);
    }

    return Object.entries(groups)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([folder, list]) => ({
        folder,
        files: list.sort((a, b) => a.path.localeCompare(b.path)),
      }));
  }, [filteredFiles]);

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="border-b border-border bg-card/30 p-4">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold">Stub Files</h1>
            <Badge variant="secondary" className="font-mono">
              {files.length} files
            </Badge>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={fetchFiles}
              disabled={loading}
              className="gap-2"
            >
              <RefreshCw
                className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
              />
              Refresh
            </Button>
          </div>
        </div>

        <div className="flex flex-col md:flex-row md:items-center gap-4 mt-4">
          <div className="relative flex-1 max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              placeholder="Search files..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9 bg-background"
            />
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 flex overflow-hidden">
        {/* File list */}
        <div className="w-full md:w-1/2 border-r border-border overflow-auto scrollbar-thin p-4">
          {loading ? (
            <div className="space-y-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <Card key={i}>
                  <CardContent className="p-3 flex items-center gap-3">
                    <Skeleton className="h-8 w-8 rounded-md" />
                    <div className="flex-1 space-y-2">
                      <Skeleton className="h-4 w-3/4" />
                      <Skeleton className="h-3 w-1/2" />
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : groupedFiles.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-full text-center p-8">
              <div className="w-16 h-16 rounded-full bg-muted flex items-center justify-center mb-4">
                <FileText className="h-8 w-8 text-muted-foreground" />
              </div>
              <h3 className="text-lg font-medium mb-2">No files found</h3>
              <p className="text-muted-foreground max-w-md">
                {searchQuery
                  ? "No files match your search criteria."
                  : "Place stub/config files into the config folder to see them here."}
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {groupedFiles.map(({ folder, files: folderFiles }) => (
                <div key={folder}>
                  <button
                    type="button"
                    onClick={() =>
                      setOpenFolders((prev) => ({
                        ...prev,
                        [folder]: !(prev[folder] ?? true),
                      }))
                    }
                    className="w-full flex items-center justify-between mb-2 hover:bg-muted/60 px-2 py-1 rounded-md transition-colors"
                  >
                    <div className="flex items-center gap-2">
                      <Folder className="h-4 w-4 text-muted-foreground" />
                      <span className="text-sm font-semibold text-muted-foreground">
                        {folder === "/" ? "Root" : folder}
                      </span>
                    </div>
                    <Badge variant="outline" className="font-mono text-xs">
                      {folderFiles.length} files
                    </Badge>
                  </button>

                  {(openFolders[folder] ?? false) && (
                    <div className="space-y-1">
                      {folderFiles.map((file) => {
                        const showMockBadge = file.isMock && !file.valid;
                        return (
                          <button
                            key={file.path}
                            type="button"
                            onClick={() => fetchContent(file)}
                            className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-left text-sm hover:bg-muted transition-colors ${
                              selectedFile?.path === file.path
                                ? "bg-muted"
                                : ""
                            }`}
                          >
                            <span className="flex items-center gap-2 min-w-0">
                              <FileText className="h-4 w-4 text-muted-foreground flex-shrink-0" />
                              <span className="truncate">{file.path}</span>
                              {showMockBadge && (
                                <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                                  isMockConfig
                                </Badge>
                              )}
                            </span>
                            <span className="text-xs text-muted-foreground ml-2 flex-shrink-0">
                              {file.size} B
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* File content */}
        <div className="hidden md:flex md:w-1/2 flex-col overflow-hidden p-4">
          <div className="flex items-center justify-between mb-3">
            <div>
              <h2 className="text-sm font-semibold">
                {selectedFile ? selectedFile.path : "No file selected"}
              </h2>
              {selectedFile && (
                <p className="text-xs text-muted-foreground">
                  {selectedFile.size} bytes
                </p>
              )}
            </div>
            {selectedFile && (
              <Button
                size="sm"
                variant="outline"
                onClick={handleSave}
                disabled={saving || contentLoading}
                className="gap-2"
              >
                {saving && (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                )}
                Save
              </Button>
            )}
          </div>

          <Card className="flex-1 overflow-hidden">
            <CardContent className="p-0 h-full">
              {contentLoading ? (
                <div className="p-4 space-y-2">
                  {Array.from({ length: 10 }).map((_, i) => (
                    <Skeleton key={i} className="h-3 w-full" />
                  ))}
                </div>
              ) : fileContent ? (
                <div className="h-full w-full flex flex-col">
                  <div className="flex-1 flex flex-col">
                    <Editor
                      height="100%"
                      language={editorLanguage}
                      theme="vs-dark"
                      value={editedContent}
                      onChange={(value) => setEditedContent(value ?? "")}
                      options={{
                        minimap: { enabled: false },
                        scrollBeyondLastLine: false,
                        automaticLayout: true,
                        fontSize: 12,
                        wordWrap: "on",
                      }}
                    />
                  </div>
                </div>
              ) : (
                <div className="h-full flex items-center justify-center text-sm text-muted-foreground">
                  Select a file to view its contents.
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

