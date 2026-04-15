import React from 'react';
import { motion } from 'framer-motion';
import { Box, Settings, Cpu, Server, CheckCircle, Rocket, ChevronRight, ChevronLeft, Terminal, HardDrive, MemoryStick } from 'lucide-react';
import { useAppStore } from '../store';
import { WizardStep } from '../types';
import ModelWizard from './ModelWizard';
import { DockerWizard } from './DockerWizard';
import { VLLMWizard } from './VLLMWizard';
import { ServerWizard } from './ServerWizard';
import ReviewWizard from './ReviewWizard';
import { DeployProgress } from './DeployProgress';

const steps: { id: WizardStep; label: string; icon: React.ElementType; desc: string }[] = [
  { id: 'model', label: 'MODEL', icon: Box, desc: '模型配置' },
  { id: 'docker', label: 'DOCKER', icon: Settings, desc: '容器配置' },
  { id: 'vllm', label: 'VLLM', icon: Cpu, desc: '推理参数' },
  { id: 'servers', label: 'NODES', icon: Server, desc: '节点配置' },
  { id: 'review', label: 'REVIEW', icon: CheckCircle, desc: '配置确认' },
  { id: 'deploy', label: 'DEPLOY', icon: Rocket, desc: '部署执行' },
];

export const DeployWizard: React.FC = () => {
  const { wizard, setWizardStep, updateWizardConfig, completeWizardStep, resetWizard } = useAppStore();
  const currentStepIndex = steps.findIndex((s) => s.id === wizard.currentStep);

  const handleVLLMChange = (config: any) => updateWizardConfig({ vllm: config });

  const renderStepContent = () => {
    switch (wizard.currentStep) {
      case 'model': return <ModelWizard />;
      case 'docker': return <DockerWizard />;
      case 'vllm':
        return (
          <div className="max-w-5xl mx-auto">
            <div className="bg-slate-900/80 border border-slate-700/50 rounded-lg p-6 mb-4">
              <div className="flex items-center gap-3 mb-6 border-b border-slate-700/50 pb-4">
                <Cpu className="w-5 h-5 text-cyan-400" />
                <h2 className="text-lg font-mono font-semibold text-slate-200">VLLM_PARAMS</h2>
                <span className="text-xs text-slate-500 font-mono ml-auto">v0.4.0</span>
              </div>
              <VLLMWizard config={wizard.config.vllm || {}} onChange={handleVLLMChange} />
            </div>
            <div className="flex justify-between">
              <button onClick={() => setWizardStep('docker')} className="px-4 py-2 bg-slate-800 border border-slate-700 text-slate-300 rounded hover:bg-slate-700 font-mono text-sm transition-colors">
                &lt; BACK
              </button>
              <button onClick={() => { completeWizardStep('vllm'); setWizardStep('servers'); }} className="px-4 py-2 bg-cyan-600 text-white rounded hover:bg-cyan-500 font-mono text-sm transition-colors">
                NEXT &gt;
              </button>
            </div>
          </div>
        );
      case 'servers':
        return (
          <div className="max-w-5xl mx-auto">
            <div className="bg-slate-900/80 border border-slate-700/50 rounded-lg p-6 mb-4">
              <ServerWizard />
            </div>
            <div className="flex justify-between">
              <button onClick={() => setWizardStep('vllm')} className="px-4 py-2 bg-slate-800 border border-slate-700 text-slate-300 rounded hover:bg-slate-700 font-mono text-sm transition-colors">
                &lt; BACK
              </button>
              <button onClick={() => { completeWizardStep('servers'); setWizardStep('review'); }} disabled={!wizard.config.servers?.length} className="px-4 py-2 bg-cyan-600 text-white rounded hover:bg-cyan-500 disabled:opacity-50 font-mono text-sm transition-colors">
                NEXT &gt;
              </button>
            </div>
          </div>
        );
      case 'review':
        return (
          <div className="max-w-5xl mx-auto">
            <div className="bg-slate-900/80 border border-slate-700/50 rounded-lg p-6 mb-4">
              <ReviewWizard />
            </div>
            <div className="flex justify-start">
              <button onClick={() => setWizardStep('servers')} className="px-4 py-2 bg-slate-800 border border-slate-700 text-slate-300 rounded hover:bg-slate-700 font-mono text-sm transition-colors">
                &lt; BACK
              </button>
            </div>
          </div>
        );
      case 'deploy':
        return (
          <div className="max-w-5xl mx-auto">
            <div className="bg-slate-900/80 border border-slate-700/50 rounded-lg p-6">
              <div className="text-center mb-8">
                <motion.div animate={{ rotate: 360 }} transition={{ duration: 2, repeat: Infinity, ease: 'linear' }} className="inline-block">
                  <Rocket className="w-12 h-12 text-cyan-400" />
                </motion.div>
                <h2 className="text-xl font-mono font-bold text-slate-200 mt-4">DEPLOYING...</h2>
                <p className="text-slate-500 font-mono text-sm mt-2">Initializing deployment pipeline</p>
              </div>
              <DeployProgress tasks={[]} />
            </div>
            <div className="flex justify-center mt-4">
              <button onClick={resetWizard} className="px-4 py-2 bg-slate-800 border border-slate-700 text-slate-300 rounded hover:bg-slate-700 font-mono text-sm transition-colors">
                RETURN
              </button>
            </div>
          </div>
        );
      default: return null;
    }
  };

  return (
    <div className="p-4">
      <div className="mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Terminal className="w-4 h-4 text-cyan-400" />
          <span className="text-xs font-mono text-slate-500">DEPLOYMENT_WIZARD_V1.0</span>
        </div>
        <div className="flex items-center">
          {steps.map((step, index) => {
            const Icon = step.icon;
            const isActive = index === currentStepIndex;
            const isCompleted = wizard.completedSteps.includes(step.id);
            return (
              <React.Fragment key={step.id}>
                <button onClick={() => (isCompleted || index <= currentStepIndex) && setWizardStep(step.id)} className="group relative">
                  <div className={`flex flex-col items-center min-w-[80px] ${isActive ? 'opacity-100' : 'opacity-60'}`}>
                    <div className={`w-10 h-10 rounded flex items-center justify-center border transition-all ${
                      isActive ? 'bg-cyan-500/20 border-cyan-500 text-cyan-400' :
                      isCompleted ? 'bg-green-500/20 border-green-500 text-green-400' :
                      'bg-slate-800 border-slate-700 text-slate-500'
                    }`}>
                      <Icon className="w-4 h-4" />
                    </div>
                    <span className={`text-[10px] font-mono mt-1 ${isActive ? 'text-cyan-400' : isCompleted ? 'text-green-400' : 'text-slate-500'}`}>{step.label}</span>
                    <span className="text-[9px] text-slate-600">{step.desc}</span>
                  </div>
                </button>
                {index < steps.length - 1 && (
                  <div className={`w-8 h-px mx-1 ${index < currentStepIndex ? 'bg-green-500' : 'bg-slate-700'}`} />
                )}
              </React.Fragment>
            );
          })}
        </div>
      </div>
      <motion.div key={wizard.currentStep} initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.2 }}>
        {renderStepContent()}
      </motion.div>
    </div>
  );
};
