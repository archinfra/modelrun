import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Database, Layers, FileCode, HardDrive, ChevronRight, ChevronDown, Hash, Binary, Scale } from 'lucide-react';
import { useAppStore } from '../store';
import { ModelConfig, ModelFile } from '../types';

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const mockModels: ModelConfig[] = [
  {
    id: '1',
    name: 'Qwen2-72B-Instruct',
    source: 'modelscope',
    modelId: 'qwen/Qwen2-72B-Instruct',
    revision: 'v1.0.0',
    size: 144394567680,
    format: 'safetensors',
    parameters: '72B',
    quantization: 'FP16',
    files: [
      { name: 'config.json', size: 2048, path: 'config.json' },
      { name: 'model-00001-of-00037.safetensors', size: 3892314112, path: 'model-00001-of-00037.safetensors' },
      { name: 'model-00002-of-00037.safetensors', size: 3892314112, path: 'model-00002-of-00037.safetensors' },
      { name: 'tokenizer.json', size: 134217728, path: 'tokenizer.json' },
    ]
  },
  {
    id: '2',
    name: 'Llama-3-8B-AWQ',
    source: 'huggingface',
    modelId: 'casperhansen/llama-3-8b-instruct-awq',
    revision: 'main',
    size: 5153960755,
    format: 'awq',
    parameters: '8B',
    quantization: 'AWQ-4bit',
    files: [
      { name: 'model.safetensors', size: 5153960752, path: 'model.safetensors' },
      { name: 'config.json', size: 1024, path: 'config.json' },
    ]
  }
];

export const ModelManager: React.FC = () => {
  const { models, addModel } = useAppStore();
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [selectedFile, setSelectedFile] = useState<ModelFile | null>(null);

  const displayModels = models.length > 0 ? models : mockModels;

  return (
    <div className="p-6 font-mono">
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          <Database className="w-6 h-6 text-cyan-400" />
          <h2 className="text-xl font-bold text-slate-100">MODEL_REGISTRY</h2>
          <span className="px-2 py-0.5 bg-cyan-500/20 text-cyan-400 text-xs rounded">{displayModels.length} ENTRIES</span>
        </div>
      </div>

      <div className="grid gap-4">
        {displayModels.map((model) => (
          <motion.div
            key={model.id}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="bg-slate-900/80 border border-slate-700 rounded-lg overflow-hidden"
          >
            <div
              className="p-4 flex items-center justify-between cursor-pointer hover:bg-slate-800/50 transition-colors"
              onClick={() => setExpandedId(expandedId === model.id ? null : model.id)}
            >
              <div className="flex items-center gap-4">
                <button className="text-slate-500">
                  {expandedId === model.id ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
                </button>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-cyan-400 font-semibold">{model.name}</span>
                    <span className="px-1.5 py-0.5 bg-slate-700 text-slate-400 text-xs rounded">{model.source}</span>
                    {model.quantization && (
                      <span className="px-1.5 py-0.5 bg-amber-500/20 text-amber-400 text-xs rounded">{model.quantization}</span>
                    )}
                  </div>
                  <div className="text-xs text-slate-500 mt-1 font-mono">{model.modelId || model.localPath}</div>
                </div>
              </div>

              <div className="flex items-center gap-6 text-sm">
                <div className="flex items-center gap-2 text-slate-400">
                  <Layers className="w-4 h-4" />
                  <span>{model.parameters}</span>
                </div>
                <div className="flex items-center gap-2 text-slate-400">
                  <HardDrive className="w-4 h-4" />
                  <span>{formatBytes(model.size || 0)}</span>
                </div>
                <div className="flex items-center gap-2 text-slate-400">
                  <FileCode className="w-4 h-4" />
                  <span>{model.format}</span>
                </div>
              </div>
            </div>

            <AnimatePresence>
              {expandedId === model.id && (
                <motion.div
                  initial={{ height: 0 }}
                  animate={{ height: 'auto' }}
                  exit={{ height: 0 }}
                  className="border-t border-slate-700 bg-slate-950/50"
                >
                  <div className="p-4 grid grid-cols-2 gap-4">
                    <div>
                      <h4 className="text-xs text-slate-500 mb-2 flex items-center gap-2">
                        <Hash className="w-3 h-3" /> FILE_MANIFEST
                      </h4>
                      <div className="space-y-1">
                        {model.files?.map((file) => (
                          <div
                            key={file.name}
                            onClick={() => setSelectedFile(file)}
                            className={`flex items-center justify-between px-3 py-2 rounded text-xs cursor-pointer transition-colors ${
                              selectedFile?.name === file.name
                                ? 'bg-cyan-500/20 text-cyan-400'
                                : 'hover:bg-slate-800 text-slate-400'
                            }`}
                          >
                            <span className="font-mono truncate">{file.name}</span>
                            <span className="text-slate-500">{formatBytes(file.size)}</span>
                          </div>
                        ))}
                      </div>
                    </div>

                    <div>
                      <h4 className="text-xs text-slate-500 mb-2 flex items-center gap-2">
                        <Binary className="w-3 h-3" /> METADATA
                      </h4>
                      <div className="space-y-2 text-xs">
                        <div className="flex justify-between py-1 border-b border-slate-800">
                          <span className="text-slate-500">Format</span>
                          <span className="text-slate-300">{model.format}</span>
                        </div>
                        <div className="flex justify-between py-1 border-b border-slate-800">
                          <span className="text-slate-500">Parameters</span>
                          <span className="text-slate-300">{model.parameters}</span>
                        </div>
                        <div className="flex justify-between py-1 border-b border-slate-800">
                          <span className="text-slate-500">Quantization</span>
                          <span className="text-slate-300">{model.quantization || 'None'}</span>
                        </div>
                        <div className="flex justify-between py-1 border-b border-slate-800">
                          <span className="text-slate-500">Revision</span>
                          <span className="text-slate-300">{model.revision}</span>
                        </div>
                        <div className="flex justify-between py-1 border-b border-slate-800">
                          <span className="text-slate-500">Total Size</span>
                          <span className="text-slate-300">{formatBytes(model.size || 0)}</span>
                        </div>
                        <div className="flex justify-between py-1">
                          <span className="text-slate-500">File Count</span>
                          <span className="text-slate-300">{model.files?.length || 0}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>
          </motion.div>
        ))}
      </div>
    </div>
  );
};
