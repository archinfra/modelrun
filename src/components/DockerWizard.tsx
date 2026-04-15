import React, { useState } from 'react';
import { motion } from 'framer-motion';
import { Box, Cpu, Database, Settings, Plus, Trash2 } from 'lucide-react';
import { useAppStore } from '../store';
import { DockerConfig } from '../types';

const commonImages = [
  'vllm/vllm-openai:latest',
  'vllm/vllm-openai:v0.4.0',
  'mindie/mindie-server:latest',
  'mindie/mindie-server:1.0.0',
];

export const DockerWizard: React.FC = () => {
  const { wizard, updateWizardConfig, setWizardStep, completeWizardStep } = useAppStore();
  const [envVars, setEnvVars] = useState<{ key: string; value: string }[]>(
    Object.entries(wizard.config.docker?.environmentVars || {}).map(([key, value]) => ({
      key,
      value,
    }))
  );
  const [volumes, setVolumes] = useState<{ host: string; container: string }[]>(
    wizard.config.docker?.volumes || []
  );

  const [formData, setFormData] = useState<DockerConfig>({
    image: wizard.config.docker?.image || 'vllm/vllm-openai:latest',
    registry: wizard.config.docker?.registry || '',
    tag: wizard.config.docker?.tag || 'latest',
    gpuDevices: wizard.config.docker?.gpuDevices || 'all',
    shmSize: wizard.config.docker?.shmSize || '16g',
    environmentVars: wizard.config.docker?.environmentVars || {},
    volumes: wizard.config.docker?.volumes || [],
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const envVarsObj = envVars.reduce((acc, { key, value }) => {
      if (key) acc[key] = value;
      return acc;
    }, {} as Record<string, string>);

    updateWizardConfig({
      docker: {
        ...formData,
        environmentVars: envVarsObj,
        volumes,
      },
    });
    completeWizardStep('docker');
    setWizardStep('vllm');
  };

  const addEnvVar = () => setEnvVars([...envVars, { key: '', value: '' }]);
  const removeEnvVar = (index: number) => setEnvVars(envVars.filter((_, i) => i !== index));
  const updateEnvVar = (index: number, field: 'key' | 'value', value: string) => {
    const newEnvVars = [...envVars];
    newEnvVars[index][field] = value;
    setEnvVars(newEnvVars);
  };

  const addVolume = () => setVolumes([...volumes, { host: '', container: '' }]);
  const removeVolume = (index: number) => setVolumes(volumes.filter((_, i) => i !== index));
  const updateVolume = (index: number, field: 'host' | 'container', value: string) => {
    const newVolumes = [...volumes];
    newVolumes[index][field] = value;
    setVolumes(newVolumes);
  };

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="max-w-4xl mx-auto"
    >
      <div className="bg-slate-800 rounded-xl p-6 border border-slate-700">
        <div className="flex items-center gap-3 mb-6">
          <Box className="w-6 h-6 text-blue-400" />
          <h2 className="text-xl font-semibold text-white">Docker配置</h2>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="col-span-2">
              <label className="block text-sm font-medium text-slate-300 mb-2">
                Docker镜像
              </label>
              <select
                value={formData.image}
                onChange={(e) => setFormData({ ...formData, image: e.target.value })}
                className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500"
              >
                {commonImages.map((img) => (
                  <option key={img} value={img}>{img}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                镜像仓库 (可选)
              </label>
              <input
                type="text"
                value={formData.registry}
                onChange={(e) => setFormData({ ...formData, registry: e.target.value })}
                placeholder="如: registry.cn-hangzhou.aliyuncs.com"
                className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                标签
              </label>
              <input
                type="text"
                value={formData.tag}
                onChange={(e) => setFormData({ ...formData, tag: e.target.value })}
                className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                <Cpu className="w-4 h-4 inline mr-1" />
                GPU设备
              </label>
              <input
                type="text"
                value={formData.gpuDevices}
                onChange={(e) => setFormData({ ...formData, gpuDevices: e.target.value })}
                placeholder="all 或 0,1,2,3"
                className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-slate-300 mb-2">
                <Database className="w-4 h-4 inline mr-1" />
                共享内存大小
              </label>
              <input
                type="text"
                value={formData.shmSize}
                onChange={(e) => setFormData({ ...formData, shmSize: e.target.value })}
                placeholder="16g"
                className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="border-t border-slate-700 pt-4">
            <div className="flex items-center justify-between mb-3">
              <label className="text-sm font-medium text-slate-300">
                <Settings className="w-4 h-4 inline mr-1" />
                环境变量
              </label>
              <button
                type="button"
                onClick={addEnvVar}
                className="flex items-center gap-1 text-sm text-blue-400 hover:text-blue-300"
              >
                <Plus className="w-4 h-4" />
                添加
              </button>
            </div>
            {envVars.map((env, index) => (
              <div key={index} className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={env.key}
                  onChange={(e) => updateEnvVar(index, 'key', e.target.value)}
                  placeholder="变量名"
                  className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm"
                />
                <input
                  type="text"
                  value={env.value}
                  onChange={(e) => updateEnvVar(index, 'value', e.target.value)}
                  placeholder="值"
                  className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm"
                />
                <button
                  type="button"
                  onClick={() => removeEnvVar(index)}
                  className="p-2 text-red-400 hover:text-red-300"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>

          <div className="border-t border-slate-700 pt-4">
            <div className="flex items-center justify-between mb-3">
              <label className="text-sm font-medium text-slate-300">数据卷挂载</label>
              <button
                type="button"
                onClick={addVolume}
                className="flex items-center gap-1 text-sm text-blue-400 hover:text-blue-300"
              >
                <Plus className="w-4 h-4" />
                添加
              </button>
            </div>
            {volumes.map((vol, index) => (
              <div key={index} className="flex gap-2 mb-2">
                <input
                  type="text"
                  value={vol.host}
                  onChange={(e) => updateVolume(index, 'host', e.target.value)}
                  placeholder="主机路径"
                  className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm"
                />
                <input
                  type="text"
                  value={vol.container}
                  onChange={(e) => updateVolume(index, 'container', e.target.value)}
                  placeholder="容器路径"
                  className="flex-1 px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm"
                />
                <button
                  type="button"
                  onClick={() => removeVolume(index)}
                  className="p-2 text-red-400 hover:text-red-300"
                >
                  <Trash2 className="w-4 h-4" />
                </button>
              </div>
            ))}
          </div>

          <div className="flex justify-between pt-4">
            <button
              type="button"
              onClick={() => setWizardStep('model')}
              className="px-6 py-2 bg-slate-700 text-white rounded-lg hover:bg-slate-600 transition-colors"
            >
              上一步
            </button>
            <button
              type="submit"
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-500 transition-colors"
            >
              下一步
            </button>
          </div>
        </form>
      </div>
    </motion.div>
  );
};
