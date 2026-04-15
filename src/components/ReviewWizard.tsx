import React from 'react';
import { motion } from 'framer-motion';
import { CheckCircle, Edit3, Server, Box, Cpu, Settings } from 'lucide-react';
import { useAppStore } from '../store';
import { WizardStep } from '../types';

const ReviewWizard: React.FC = () => {
  const { wizard, setWizardStep, completeWizardStep } = useAppStore();
  const { config } = wizard;

  const handleDeploy = () => {
    completeWizardStep('review');
    setWizardStep('deploy');
  };

  const handleEdit = (step: WizardStep) => {
    setWizardStep(step);
  };

  const sections = [
    {
      id: 'model' as WizardStep,
      title: '模型配置',
      icon: Box,
      content: config.model && (
        <div className="space-y-1 text-sm">
          <p><span className="text-gray-400">名称:</span> {config.model.name}</p>
          <p><span className="text-gray-400">来源:</span> {config.model.source}</p>
          <p><span className="text-gray-400">模型ID:</span> {config.model.modelId}</p>
        </div>
      ),
    },
    {
      id: 'docker' as WizardStep,
      title: 'Docker配置',
      icon: Settings,
      content: config.docker && (
        <div className="space-y-1 text-sm">
          <p><span className="text-gray-400">镜像:</span> {config.docker.image}:{config.docker.tag}</p>
          <p><span className="text-gray-400">GPU:</span> {config.docker.gpuDevices}</p>
          <p><span className="text-gray-400">端口:</span> {config.apiPort}</p>
        </div>
      ),
    },
    {
      id: 'vllm' as WizardStep,
      title: 'vLLM参数',
      icon: Cpu,
      content: config.vllm && (
        <div className="space-y-1 text-sm">
          <p><span className="text-gray-400">TP:</span> {config.vllm.tensorParallelSize}</p>
          <p><span className="text-gray-400">PP:</span> {config.vllm.pipelineParallelSize}</p>
          <p><span className="text-gray-400">量化:</span> {config.vllm.quantization || '无'}</p>
        </div>
      ),
    },
    {
      id: 'servers' as WizardStep,
      title: '服务器配置',
      icon: Server,
      content: config.servers && (
        <div className="space-y-1 text-sm">
          <p><span className="text-gray-400">服务器数:</span> {config.servers.length}台</p>
          <p className="text-gray-400">服务器列表:</p>
          <ul className="pl-4 text-xs text-gray-500">
            {config.servers.map((id) => (
              <li key={id}>{id}</li>
            ))}
          </ul>
        </div>
      ),
    },
  ];

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="space-y-6"
    >
      <div className="text-center mb-8">
        <CheckCircle className="w-16 h-16 text-green-500 mx-auto mb-4" />
        <h2 className="text-2xl font-bold text-white">配置确认</h2>
        <p className="text-gray-400 mt-2">请确认以下配置信息，确认无误后开始部署</p>
      </div>

      <div className="grid grid-cols-2 gap-4">
        {sections.map((section) => (
          <motion.div
            key={section.id}
            whileHover={{ scale: 1.02 }}
            className="bg-gray-800/50 rounded-lg p-4 border border-gray-700"
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <section.icon className="w-5 h-5 text-blue-400" />
                <h3 className="font-semibold text-white">{section.title}</h3>
              </div>
              <button
                onClick={() => handleEdit(section.id)}
                className="p-1 hover:bg-gray-700 rounded transition-colors"
              >
                <Edit3 className="w-4 h-4 text-gray-400" />
              </button>
            </div>
            {section.content || (
              <p className="text-sm text-gray-500">未配置</p>
            )}
          </motion.div>
        ))}
      </div>

      <div className="flex justify-center pt-6">
        <motion.button
          whileHover={{ scale: 1.05 }}
          whileTap={{ scale: 0.95 }}
          onClick={handleDeploy}
          className="px-8 py-3 bg-gradient-to-r from-green-500 to-emerald-600 text-white rounded-lg font-semibold shadow-lg shadow-green-500/25"
        >
          开始部署
        </motion.button>
      </div>
    </motion.div>
  );
};

export default ReviewWizard;
