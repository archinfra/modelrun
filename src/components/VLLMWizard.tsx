import React from 'react';
import { VLLMParams } from '../types';

interface VLLMWizardProps {
  config: Partial<VLLMParams>;
  onChange: (config: VLLMParams) => void;
}

const defaults: VLLMParams = {
  tensorParallelSize: 1,
  pipelineParallelSize: 1,
  maxModelLen: 4096,
  gpuMemoryUtilization: 0.9,
  dtype: 'auto',
  trustRemoteCode: true,
  enablePrefixCaching: true,
  maxNumSeqs: 256,
  maxNumBatchedTokens: 8192,
};

const numberFields: Array<{
  key: keyof Pick<
    VLLMParams,
    | 'tensorParallelSize'
    | 'pipelineParallelSize'
    | 'maxModelLen'
    | 'gpuMemoryUtilization'
    | 'maxNumSeqs'
    | 'maxNumBatchedTokens'
    | 'swapSpace'
    | 'numSpeculativeTokens'
  >;
  label: string;
  min: number;
  step?: number;
}> = [
  { key: 'tensorParallelSize', label: 'Tensor Parallel', min: 1 },
  { key: 'pipelineParallelSize', label: 'Pipeline Parallel', min: 1 },
  { key: 'maxModelLen', label: 'Max Model Len', min: 1 },
  { key: 'gpuMemoryUtilization', label: 'GPU Memory Utilization', min: 0, step: 0.01 },
  { key: 'maxNumSeqs', label: 'Max Num Seqs', min: 1 },
  { key: 'maxNumBatchedTokens', label: 'Max Batched Tokens', min: 1 },
  { key: 'swapSpace', label: 'Swap Space', min: 0 },
  { key: 'numSpeculativeTokens', label: 'Speculative Tokens', min: 0 },
];

export const VLLMWizard: React.FC<VLLMWizardProps> = ({ config, onChange }) => {
  const value: VLLMParams = { ...defaults, ...config };

  const update = <K extends keyof VLLMParams>(key: K, next: VLLMParams[K]) => {
    onChange({ ...value, [key]: next });
  };

  return (
    <div className="grid grid-cols-2 gap-4">
      {numberFields.map((field) => (
        <label key={field.key} className="space-y-2">
          <span className="block text-xs font-mono text-slate-400">{field.label}</span>
          <input
            type="number"
            min={field.min}
            step={field.step || 1}
            value={(value[field.key] as number | undefined) ?? ''}
            onChange={(event) => update(field.key, Number(event.target.value) as never)}
            className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded text-slate-200 font-mono text-sm focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
          />
        </label>
      ))}

      <label className="space-y-2">
        <span className="block text-xs font-mono text-slate-400">Dtype</span>
        <select
          value={value.dtype}
          onChange={(event) => update('dtype', event.target.value)}
          className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded text-slate-200 font-mono text-sm focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
        >
          <option value="auto">auto</option>
          <option value="float16">float16</option>
          <option value="bfloat16">bfloat16</option>
          <option value="float32">float32</option>
        </select>
      </label>

      <label className="space-y-2">
        <span className="block text-xs font-mono text-slate-400">Quantization</span>
        <input
          type="text"
          value={value.quantization || ''}
          onChange={(event) => update('quantization', event.target.value)}
          placeholder="awq / gptq / fp8"
          className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded text-slate-200 font-mono text-sm focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
        />
      </label>

      <label className="col-span-2 space-y-2">
        <span className="block text-xs font-mono text-slate-400">Speculative Model</span>
        <input
          type="text"
          value={value.speculativeModel || ''}
          onChange={(event) => update('speculativeModel', event.target.value)}
          placeholder="optional draft model"
          className="w-full px-3 py-2 bg-slate-800 border border-slate-700 rounded text-slate-200 font-mono text-sm focus:ring-2 focus:ring-cyan-500 focus:border-transparent"
        />
      </label>

      {[
        ['trustRemoteCode', 'Trust Remote Code'],
        ['enablePrefixCaching', 'Enable Prefix Caching'],
        ['enforceEager', 'Enforce Eager'],
        ['enableChunkedPrefill', 'Enable Chunked Prefill'],
      ].map(([key, label]) => (
        <label key={key} className="flex items-center gap-2 text-sm text-slate-300">
          <input
            type="checkbox"
            checked={Boolean(value[key as keyof VLLMParams])}
            onChange={(event) => update(key as keyof VLLMParams, event.target.checked as never)}
            className="rounded border-slate-600 bg-slate-800 text-cyan-500 focus:ring-cyan-500"
          />
          {label}
        </label>
      ))}
    </div>
  );
};
