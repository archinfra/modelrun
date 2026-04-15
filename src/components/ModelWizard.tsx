import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Database, Download, Folder, ChevronRight } from 'lucide-react';
import { useAppStore } from '../store';
import { ModelConfig } from '../types';

const modelSources = [
  { id: 'modelscope', name: 'ModelScope', icon: Database, desc: '从魔搭社区下载模型' },
  { id: 'huggingface', name: 'HuggingFace', icon: Download, desc: '从HuggingFace下载模型' },
  { id: 'local', name: '本地模型', icon: Folder, desc: '使用服务器上的本地模型' },
];

export default function ModelWizard() {
  const { wizard, updateWizardConfig, setWizardStep, completeWizardStep } = useAppStore();
  const [formData, setFormData] = useState<Partial<ModelConfig>>({
    source: 'modelscope',
    modelId: '',
    revision: 'main',
    localPath: '',
  });

  const handleNext = () => {
    if (!formData.modelId && formData.source !== 'local') return;
    if (formData.source === 'local' && !formData.localPath) return;

    const model: ModelConfig = {
      id: Date.now().toString(),
      name: formData.modelId || formData.localPath || '未命名模型',
      source: formData.source as 'modelscope' | 'huggingface' | 'local',
      modelId: formData.modelId || '',
      revision: formData.revision,
      localPath: formData.localPath,
    };

    updateWizardConfig({ model });
    completeWizardStep('model');
    setWizardStep('docker');
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="max-w-4xl mx-auto p-6"
    >
      <h2 className="text-2xl font-bold text-gray-900 mb-2">选择模型</h2>
      <p className="text-gray-500 mb-8">配置要部署的模型来源和版本</p>

      <div className="grid grid-cols-3 gap-4 mb-8">
        {modelSources.map((source) => (
          <button
            key={source.id}
            onClick={() => setFormData({ ...formData, source: source.id as any })}
            className={`p-6 rounded-xl border-2 transition-all text-left ${
              formData.source === source.id
                ? 'border-blue-500 bg-blue-50'
                : 'border-gray-200 hover:border-gray-300'
            }`}
          >
            <source.icon className="w-8 h-8 mb-3 text-blue-600" />
            <h3 className="font-semibold text-gray-900">{source.name}</h3>
            <p className="text-sm text-gray-500 mt-1">{source.desc}</p>
          </button>
        ))}
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-6">
        {formData.source !== 'local' ? (
          <>
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                模型ID
              </label>
              <input
                type="text"
                value={formData.modelId}
                onChange={(e) => setFormData({ ...formData, modelId: e.target.value })}
                placeholder={formData.source === 'modelscope' ? '如: llm-research/llama-2-7b' : '如: meta-llama/Llama-2-7b'}
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                版本/分支
              </label>
              <input
                type="text"
                value={formData.revision}
                onChange={(e) => setFormData({ ...formData, revision: e.target.value })}
                placeholder="main"
                className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
          </>
        ) : (
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-2">
              本地模型路径
            </label>
            <input
              type="text"
              value={formData.localPath}
              onChange={(e) => setFormData({ ...formData, localPath: e.target.value })}
              placeholder="如: /data/models/llama-2-7b"
              className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            />
          </div>
        )}
      </div>

      <div className="flex justify-end mt-8">
        <button
          onClick={handleNext}
          disabled={!formData.modelId && formData.source !== 'local'}
          className="flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
        >
          下一步
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </motion.div>
  );
}
