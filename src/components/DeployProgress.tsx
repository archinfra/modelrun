import React, { useRef, useEffect } from 'react';
import { motion } from 'framer-motion';
import { Terminal, CheckCircle, XCircle, Loader2, Clock } from 'lucide-react';
import { DeploymentStep, DeploymentTask } from '../types';

interface DeployProgressProps {
  tasks: DeploymentTask[];
}

const StepIcon: React.FC<{ status: DeploymentStep['status'] }> = ({ status }) => {
  switch (status) {
    case 'completed':
      return <CheckCircle className="w-4 h-4 text-green-500" />;
    case 'failed':
      return <XCircle className="w-4 h-4 text-red-500" />;
    case 'running':
      return <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />;
    default:
      return <Clock className="w-4 h-4 text-gray-500" />;
  }
};

export const DeployProgress: React.FC<DeployProgressProps> = ({ tasks }) => {
  const logsEndRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [tasks]);

  const allLogs = tasks.flatMap(t =>
    t.steps.flatMap(s =>
      s.logs.map(message => ({ message, serverId: t.serverId, stepName: s.name }))
    )
  );

  return (
    <div className="space-y-4">
      {tasks.map((task) => (
        <motion.div
          key={task.id}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="bg-gray-900 rounded-lg border border-gray-800"
        >
          <div className="px-4 py-3 bg-gray-800/50 border-b border-gray-800 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono text-gray-400">[{task.serverId}]</span>
              <span className="text-sm text-gray-200">{task.overallProgress}%</span>
            </div>
            <div className="w-32 h-1.5 bg-gray-700 rounded-full">
              <motion.div
                className="h-full bg-blue-500 rounded-full"
                initial={{ width: 0 }}
                animate={{ width: `${task.overallProgress}%` }}
              />
            </div>
          </div>

          <div className="p-4 space-y-2">
            {task.steps.map((step) => (
              <div key={step.id} className="flex items-start gap-3">
                <StepIcon status={step.status} />
                <div className="flex-1">
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-gray-300">{step.name}</span>
                    <span className="text-xs text-gray-500">{step.progress}%</span>
                  </div>
                  <p className="text-xs text-gray-500">{step.description}</p>
                </div>
              </div>
            ))}
          </div>
        </motion.div>
      ))}

      <div className="bg-gray-950 rounded-lg border border-gray-800">
        <div className="px-4 py-2 bg-gray-900/50 border-b border-gray-800 flex items-center gap-2">
          <Terminal className="w-4 h-4 text-green-500" />
          <span className="text-xs text-gray-400 font-mono">DEPLOYMENT LOGS</span>
        </div>
        <div className="p-4 h-64 overflow-y-auto font-mono text-xs space-y-1">
          {allLogs.length === 0 ? (
            <span className="text-gray-600">Waiting for logs...</span>
          ) : (
            allLogs.map((log, i) => (
              <div key={i} className="text-gray-400">
                <span className="text-gray-600">[{log.serverId}]</span>
                <span className="text-blue-500 ml-2">{log.stepName}</span>
                <span className="ml-2">{log.message}</span>
              </div>
            ))
          )}
          <div ref={logsEndRef} />
        </div>
      </div>
    </div>
  );
};
