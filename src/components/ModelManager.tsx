import React, { useEffect, useState } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import {
  Binary,
  ChevronDown,
  ChevronRight,
  Database,
  FileCode,
  HardDrive,
  Info,
  Layers,
} from 'lucide-react';
import { requestJSON } from '../lib/api';
import { ModelConfig, ModelFile } from '../types';

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const index = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 2)} ${units[index]}`;
};

export const ModelManager: React.FC = () => {
  const [models, setModels] = useState<ModelConfig[]>([]);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<ModelFile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;

    requestJSON<ModelConfig[]>('/api/models')
      .then((payload) => {
        if (!active) return;
        setModels(Array.isArray(payload) ? payload : []);
        setError('');
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : 'Failed to load models');
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Model Registry</h1>
          <p className="text-slate-500 mt-1">这里只展示后端 SQLite 中已持久化的模型记录，不再回退到前端 mock 数据。</p>
        </div>
        <span className="px-3 py-1 rounded-full bg-slate-100 text-slate-700 text-sm font-medium">
          {models.length} persisted
        </span>
      </div>

      <div className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 flex gap-3">
        <Info className="w-4 h-4 mt-0.5 shrink-0" />
        <span>
          模型列表和模型明细来自后端落库数据。当前只有 <code>/api/models/search</code> 的搜索目录仍是内置演示目录，
          已创建模型本身是落到 SQLite 的。
        </span>
      </div>

      {error && (
        <div className="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
      )}

      <div className="grid gap-4">
        {models.map((model) => (
          <motion.div
            key={model.id}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            className="bg-white border border-slate-200 rounded-2xl overflow-hidden"
          >
            <button
              type="button"
              onClick={() => setExpandedId(expandedId === model.id ? null : model.id)}
              className="w-full p-5 flex items-start justify-between gap-4 hover:bg-slate-50 transition-colors"
            >
              <div className="flex items-start gap-4 text-left min-w-0">
                <div className="w-11 h-11 rounded-xl bg-blue-100 text-blue-600 flex items-center justify-center shrink-0">
                  <Database className="w-5 h-5" />
                </div>
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-semibold text-slate-900">{model.name}</span>
                    <span className="px-2 py-0.5 rounded-full bg-slate-100 text-slate-600 text-xs">{model.source}</span>
                    {model.quantization && (
                      <span className="px-2 py-0.5 rounded-full bg-amber-100 text-amber-700 text-xs">{model.quantization}</span>
                    )}
                  </div>
                  <div className="text-sm text-slate-500 mt-1 break-all">{model.modelId || model.localPath || '-'}</div>
                </div>
              </div>

              <div className="flex items-center gap-6 text-sm shrink-0">
                <div className="flex items-center gap-2 text-slate-500">
                  <Layers className="w-4 h-4" />
                  <span>{model.parameters || '-'}</span>
                </div>
                <div className="flex items-center gap-2 text-slate-500">
                  <HardDrive className="w-4 h-4" />
                  <span>{formatBytes(model.size || 0)}</span>
                </div>
                <div className="flex items-center gap-2 text-slate-500">
                  <FileCode className="w-4 h-4" />
                  <span>{model.format || '-'}</span>
                </div>
                {expandedId === model.id ? (
                  <ChevronDown className="w-5 h-5 text-slate-400" />
                ) : (
                  <ChevronRight className="w-5 h-5 text-slate-400" />
                )}
              </div>
            </button>

            <AnimatePresence initial={false}>
              {expandedId === model.id && (
                <motion.div
                  initial={{ height: 0, opacity: 0 }}
                  animate={{ height: 'auto', opacity: 1 }}
                  exit={{ height: 0, opacity: 0 }}
                  className="border-t border-slate-200 bg-slate-50"
                >
                  <div className="p-5 grid gap-5 lg:grid-cols-[1.2fr,0.8fr]">
                    <div>
                      <h3 className="text-sm font-medium text-slate-900 mb-3">File Manifest</h3>
                      {model.files?.length ? (
                        <div className="space-y-2">
                          {model.files.map((file) => (
                            <button
                              type="button"
                              key={`${model.id}-${file.path}`}
                              onClick={() => setSelectedFile(file)}
                              className={`w-full flex items-center justify-between gap-3 px-3 py-2 rounded-xl border text-left transition-colors ${
                                selectedFile?.path === file.path
                                  ? 'border-blue-200 bg-blue-50 text-blue-700'
                                  : 'border-slate-200 bg-white text-slate-700 hover:bg-slate-100'
                              }`}
                            >
                              <span className="truncate text-sm">{file.path}</span>
                              <span className="text-xs text-slate-500 shrink-0">{formatBytes(file.size)}</span>
                            </button>
                          ))}
                        </div>
                      ) : (
                        <div className="rounded-xl border border-dashed border-slate-200 bg-white px-4 py-6 text-sm text-slate-500">
                          没有扫描到文件清单。
                        </div>
                      )}
                    </div>

                    <div className="space-y-4">
                      <div className="rounded-2xl border border-slate-200 bg-white p-4">
                        <h3 className="text-sm font-medium text-slate-900 mb-3 flex items-center gap-2">
                          <Binary className="w-4 h-4 text-slate-500" />
                          Metadata
                        </h3>
                        <div className="space-y-2 text-sm">
                          <Row label="Source" value={model.source} />
                          <Row label="Revision" value={model.revision || '-'} />
                          <Row label="Format" value={model.format || '-'} />
                          <Row label="Parameters" value={model.parameters || '-'} />
                          <Row label="Quantization" value={model.quantization || '-'} />
                          <Row label="Size" value={formatBytes(model.size || 0)} />
                          <Row label="Files" value={`${model.files?.length || 0}`} />
                        </div>
                      </div>

                      {selectedFile && (
                        <div className="rounded-2xl border border-slate-200 bg-white p-4 text-sm">
                          <div className="font-medium text-slate-900">{selectedFile.name}</div>
                          <div className="text-slate-500 mt-1 break-all">{selectedFile.path}</div>
                          <div className="text-slate-500 mt-2">{formatBytes(selectedFile.size)}</div>
                        </div>
                      )}
                    </div>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        ))}
      </div>

      {!loading && models.length === 0 && !error && (
        <div className="text-center py-16 bg-white rounded-2xl border border-dashed border-slate-200">
          <Database className="w-14 h-14 text-slate-300 mx-auto mb-4" />
          <p className="text-slate-600 font-medium">后端还没有持久化模型记录</p>
          <p className="text-slate-500 mt-2 text-sm">添加模型后，这里显示的就是 SQLite 里的真实数据。</p>
        </div>
      )}
    </div>
  );
};

const Row: React.FC<{ label: string; value: string }> = ({ label, value }) => (
  <div className="flex items-center justify-between gap-3">
    <span className="text-slate-500">{label}</span>
    <span className="text-slate-900 text-right break-all">{value}</span>
  </div>
);
