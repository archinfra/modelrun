import React, { useEffect, useMemo, useState } from 'react';
import {
  AlertCircle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  Clock,
  Globe,
  RefreshCw,
  Server,
  Terminal,
  XCircle,
} from 'lucide-react';
import { useAppStore } from '../store';
import {
  RemoteTask,
  RemoteTaskExecutionType,
  RemoteTaskPreset,
  RemoteTaskScope,
  ServerConfig,
} from '../types';

type TaskDraft = {
  name: string;
  description: string;
  scope: RemoteTaskScope;
  executionType: RemoteTaskExecutionType;
  serverIds: string[];
  command: string;
  scriptUrl: string;
  scriptArgs: string;
  presetId: string;
  presetArgs: Record<string, string>;
};

const requestJSON = async <T,>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
  });
  const text = await response.text();
  const payload = text ? JSON.parse(text) : null;
  if (!response.ok) {
    throw new Error(payload?.error || response.statusText || 'request failed');
  }
  return payload as T;
};

const buildPresetArgs = (
  preset: RemoteTaskPreset | undefined,
  current: Record<string, string>
): Record<string, string> => {
  if (!preset?.fields?.length) return {};
  return preset.fields.reduce<Record<string, string>>((acc, field) => {
    acc[field.key] = current[field.key] ?? field.defaultValue ?? '';
    return acc;
  }, {});
};

const createEmptyDraft = (): TaskDraft => ({
  name: '',
  description: '',
  scope: 'all',
  executionType: 'preset',
  serverIds: [],
  command: 'docker ps',
  scriptUrl: '',
  scriptArgs: '',
  presetId: '',
  presetArgs: {},
});

const statusClassName = (status: string) => {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-700 border-emerald-200';
    case 'running':
      return 'bg-blue-100 text-blue-700 border-blue-200';
    case 'failed':
      return 'bg-red-100 text-red-700 border-red-200';
    case 'partial':
      return 'bg-amber-100 text-amber-700 border-amber-200';
    default:
      return 'bg-slate-100 text-slate-600 border-slate-200';
  }
};

const executionTypeLabel = (executionType: RemoteTaskExecutionType) => {
  switch (executionType) {
    case 'preset':
      return 'Preset';
    case 'script_url':
      return 'Remote Script';
    default:
      return 'Shell Command';
  }
};

const scopeLabel = (scope: RemoteTaskScope, projectName?: string) => {
  switch (scope) {
    case 'project':
      return projectName ? `Current project: ${projectName}` : 'Current project';
    case 'selected':
      return 'Selected robots';
    default:
      return 'All robots';
  }
};

export const TaskDispatchManager: React.FC = () => {
  const { projects, currentProjectId, servers, setServers } = useAppStore();
  const [tasks, setTasks] = useState<RemoteTask[]>([]);
  const [presets, setPresets] = useState<RemoteTaskPreset[]>([]);
  const [draft, setDraft] = useState<TaskDraft>(createEmptyDraft);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [expandedTaskId, setExpandedTaskId] = useState<string | null>(null);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  const currentProject = projects.find((project) => project.id === currentProjectId);
  const currentProjectServers = useMemo(
    () => servers.filter((server) => server.projectId === currentProjectId),
    [servers, currentProjectId]
  );
  const selectedPreset = useMemo(
    () => presets.find((preset) => preset.id === draft.presetId),
    [draft.presetId, presets]
  );

  const loadServers = async () => {
    const remoteServers = await requestJSON<ServerConfig[]>('/api/servers');
    setServers(Array.isArray(remoteServers) ? remoteServers : []);
  };

  const loadPresets = async () => {
    const remotePresets = await requestJSON<RemoteTaskPreset[]>('/api/remote-task-presets');
    const nextPresets = Array.isArray(remotePresets) ? remotePresets : [];
    setPresets(nextPresets);
    setDraft((current) => {
      const nextPresetId = current.presetId || nextPresets[0]?.id || '';
      const preset = nextPresets.find((item) => item.id === nextPresetId);
      const nextScope =
        current.scope === 'all' && currentProjectId
          ? 'project'
          : current.scope === 'project' && !currentProjectId
          ? 'all'
          : current.scope;
      return {
        ...current,
        scope: nextScope,
        presetId: nextPresetId,
        presetArgs: buildPresetArgs(preset, current.presetArgs),
      };
    });
  };

  const loadTasks = async () => {
    const remoteTasks = await requestJSON<RemoteTask[]>('/api/remote-tasks');
    setTasks(Array.isArray(remoteTasks) ? remoteTasks : []);
  };

  useEffect(() => {
    let active = true;

    const load = async () => {
      try {
        await Promise.all([loadServers(), loadPresets(), loadTasks()]);
        if (!active) return;
        setError('');
      } catch (err) {
        if (!active) return;
        setError(err instanceof Error ? err.message : 'Failed to load remote task data');
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    };

    void load();
    const timer = window.setInterval(() => {
      void loadTasks().catch(() => undefined);
    }, 3000);

    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [currentProjectId, setServers]);

  useEffect(() => {
    setDraft((current) => {
      if (current.scope === 'project' && !currentProjectId) {
        return { ...current, scope: 'all' };
      }
      if (current.scope === 'all' && currentProjectId && current.serverIds.length === 0) {
        return { ...current, scope: 'project' };
      }
      return current;
    });
  }, [currentProjectId]);

  const handlePresetChange = (presetId: string) => {
    const preset = presets.find((item) => item.id === presetId);
    setDraft((current) => ({
      ...current,
      presetId,
      presetArgs: buildPresetArgs(preset, current.presetArgs),
    }));
  };

  const handleServerToggle = (serverId: string) => {
    setDraft((current) => ({
      ...current,
      serverIds: current.serverIds.includes(serverId)
        ? current.serverIds.filter((item) => item !== serverId)
        : [...current.serverIds, serverId],
    }));
  };

  const validateDraft = (): string | null => {
    if (draft.scope === 'project' && !currentProjectId) {
      return 'Select a project or switch the scope to all robots.';
    }
    if (draft.scope === 'selected' && draft.serverIds.length === 0) {
      return 'Pick at least one robot for selected scope.';
    }

    switch (draft.executionType) {
      case 'command':
        return draft.command.trim() ? null : 'Shell command is required.';
      case 'script_url':
        return draft.scriptUrl.trim() ? null : 'Remote script URL is required.';
      case 'preset':
        if (!draft.presetId) {
          return 'Choose a preset first.';
        }
        if (selectedPreset?.fields?.some((field) => field.required && !draft.presetArgs[field.key]?.trim())) {
          return 'Fill in all required preset fields.';
        }
        return null;
      default:
        return 'Unsupported execution type.';
    }
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();

    const validationError = validateDraft();
    if (validationError) {
      setError(validationError);
      setNotice('');
      return;
    }

    setSubmitting(true);
    setError('');
    setNotice('');

    try {
      const created = await requestJSON<RemoteTask>('/api/remote-tasks', {
        method: 'POST',
        body: JSON.stringify({
          name: draft.name.trim(),
          description: draft.description.trim(),
          projectId: draft.scope === 'project' ? currentProjectId : '',
          scope: draft.scope,
          executionType: draft.executionType,
          command: draft.executionType === 'command' ? draft.command : '',
          scriptUrl: draft.executionType === 'script_url' ? draft.scriptUrl : '',
          scriptArgs: draft.executionType === 'script_url' ? draft.scriptArgs : '',
          presetId: draft.executionType === 'preset' ? draft.presetId : '',
          presetArgs: draft.executionType === 'preset' ? draft.presetArgs : {},
          serverIds: draft.scope === 'selected' ? draft.serverIds : [],
        }),
      });
      setNotice(`Task dispatched to ${created.serverIds.length} robot(s).`);
      setExpandedTaskId(created.id);
      await loadTasks();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to dispatch task');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Task Dispatch</h1>
          <p className="text-slate-500 mt-1">
            Dispatch shell commands, remote scripts, or presets to all robots from one place.
          </p>
        </div>
        <button
          onClick={() => void loadTasks()}
          className="flex items-center gap-2 px-4 py-2.5 bg-white border border-slate-200 rounded-xl text-slate-700 hover:bg-slate-50 transition-colors"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {(notice || error) && (
        <div
          className={`flex items-center gap-2 rounded-xl border px-4 py-3 text-sm ${
            error ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-700'
          }`}
        >
          {error ? <XCircle className="w-4 h-4" /> : <CheckCircle2 className="w-4 h-4" />}
          <span>{error || notice}</span>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[1.1fr,0.9fr]">
        <form onSubmit={handleSubmit} className="bg-white border border-slate-200 rounded-2xl p-6 space-y-5">
          <div className="flex items-center gap-3">
            <div className="w-11 h-11 rounded-xl bg-blue-100 text-blue-600 flex items-center justify-center">
              <Terminal className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Create Remote Task</h2>
              <p className="text-sm text-slate-500">Choose a scope, pick an execution mode, then dispatch.</p>
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">Task name</label>
              <input
                type="text"
                value={draft.name}
                onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
                className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                placeholder="Optional display name"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">Execution type</label>
              <select
                value={draft.executionType}
                onChange={(event) =>
                  setDraft((current) => ({
                    ...current,
                    executionType: event.target.value as RemoteTaskExecutionType,
                  }))
                }
                className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white focus:ring-2 focus:ring-blue-500"
              >
                <option value="preset">Built-in preset</option>
                <option value="command">Shell command</option>
                <option value="script_url">Remote script URL</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows={2}
              value={draft.description}
              onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))}
              className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
              placeholder="Optional context for operators"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">Scope</label>
            <div className="grid gap-3 md:grid-cols-3">
              {[
                {
                  value: 'project' as RemoteTaskScope,
                  title: 'Current project',
                  description: currentProject ? `${currentProject.name} / ${currentProjectServers.length} robots` : 'No project selected',
                  disabled: !currentProjectId,
                },
                {
                  value: 'selected' as RemoteTaskScope,
                  title: 'Selected robots',
                  description: `${draft.serverIds.length} robot(s) selected`,
                  disabled: servers.length === 0,
                },
                {
                  value: 'all' as RemoteTaskScope,
                  title: 'All robots',
                  description: `${servers.length} configured robot(s)`,
                  disabled: servers.length === 0,
                },
              ].map((option) => (
                <button
                  key={option.value}
                  type="button"
                  disabled={option.disabled}
                  onClick={() => setDraft((current) => ({ ...current, scope: option.value }))}
                  className={`rounded-2xl border p-4 text-left transition-colors ${
                    draft.scope === option.value
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-slate-200 bg-slate-50 hover:bg-white'
                  } ${option.disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                >
                  <div className="flex items-center gap-2 text-slate-900 font-medium">
                    {option.value === 'all' ? <Globe className="w-4 h-4" /> : <Server className="w-4 h-4" />}
                    {option.title}
                  </div>
                  <p className="text-sm text-slate-500 mt-1">{option.description}</p>
                </button>
              ))}
            </div>
          </div>

          {draft.scope === 'selected' && (
            <div className="border border-slate-200 rounded-2xl p-4">
              <div className="flex items-center justify-between gap-3 mb-3">
                <div>
                  <h3 className="font-medium text-slate-900">Choose robots</h3>
                  <p className="text-sm text-slate-500">Dispatch only to the checked targets.</p>
                </div>
                <span className="text-sm text-slate-500">{draft.serverIds.length} selected</span>
              </div>
              <div className="grid gap-2 max-h-64 overflow-y-auto">
                {servers.map((server) => {
                  const projectName = projects.find((project) => project.id === server.projectId)?.name || 'No project';
                  return (
                    <label
                      key={server.id}
                      className="flex items-start gap-3 p-3 rounded-xl border border-slate-200 hover:bg-slate-50"
                    >
                      <input
                        type="checkbox"
                        checked={draft.serverIds.includes(server.id)}
                        onChange={() => handleServerToggle(server.id)}
                        className="mt-1"
                      />
                      <span className="min-w-0">
                        <span className="block font-medium text-slate-900">{server.name}</span>
                        <span className="block text-sm text-slate-500">
                          {server.host} | {projectName}
                        </span>
                      </span>
                    </label>
                  );
                })}
              </div>
            </div>
          )}

          {draft.executionType === 'command' && (
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">Shell command</label>
              <textarea
                rows={6}
                value={draft.command}
                onChange={(event) => setDraft((current) => ({ ...current, command: event.target.value }))}
                className="w-full px-4 py-3 border border-slate-200 rounded-xl font-mono text-sm focus:ring-2 focus:ring-blue-500"
                placeholder="docker ps"
              />
            </div>
          )}

          {draft.executionType === 'script_url' && (
            <div className="grid gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Remote script URL</label>
                <input
                  type="url"
                  value={draft.scriptUrl}
                  onChange={(event) => setDraft((current) => ({ ...current, scriptUrl: event.target.value }))}
                  className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                  placeholder="https://example.com/install.sh"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Script args</label>
                <input
                  type="text"
                  value={draft.scriptArgs}
                  onChange={(event) => setDraft((current) => ({ ...current, scriptArgs: event.target.value }))}
                  className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                  placeholder="--channel prod --force"
                />
                <p className="text-xs text-slate-500 mt-1">Args are appended after `bash -s --` on the target robot.</p>
              </div>
            </div>
          )}

          {draft.executionType === 'preset' && (
            <div className="grid gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 mb-1">Preset</label>
                <select
                  value={draft.presetId}
                  onChange={(event) => handlePresetChange(event.target.value)}
                  className="w-full px-4 py-2 border border-slate-200 rounded-xl bg-white focus:ring-2 focus:ring-blue-500"
                >
                  {presets.map((preset) => (
                    <option key={preset.id} value={preset.id}>
                      {preset.name}
                    </option>
                  ))}
                </select>
                {selectedPreset && <p className="text-sm text-slate-500 mt-2">{selectedPreset.description}</p>}
              </div>
              {selectedPreset?.fields?.map((field) => (
                <div key={field.key}>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    {field.label}
                    {field.required ? ' *' : ''}
                  </label>
                  <input
                    type="text"
                    value={draft.presetArgs[field.key] || ''}
                    onChange={(event) =>
                      setDraft((current) => ({
                        ...current,
                        presetArgs: {
                          ...current.presetArgs,
                          [field.key]: event.target.value,
                        },
                      }))
                    }
                    className="w-full px-4 py-2 border border-slate-200 rounded-xl focus:ring-2 focus:ring-blue-500"
                    placeholder={field.placeholder || field.defaultValue || ''}
                  />
                  {field.description && <p className="text-xs text-slate-500 mt-1">{field.description}</p>}
                </div>
              ))}
            </div>
          )}

          <div className="flex items-center justify-between gap-4 pt-2">
            <div className="text-sm text-slate-500">
              Targeting{' '}
              {draft.scope === 'project'
                ? currentProjectServers.length
                : draft.scope === 'selected'
                ? draft.serverIds.length
                : servers.length}{' '}
              robot(s)
            </div>
            <button
              type="submit"
              disabled={submitting || loading}
              className="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-300 text-white rounded-xl font-medium transition-colors"
            >
              {submitting ? 'Dispatching...' : 'Dispatch task'}
            </button>
          </div>
        </form>

        <div className="bg-white border border-slate-200 rounded-2xl p-6 space-y-4">
          <div className="flex items-center gap-3">
            <div className="w-11 h-11 rounded-xl bg-slate-100 text-slate-700 flex items-center justify-center">
              <Clock className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Quick Notes</h2>
              <p className="text-sm text-slate-500">Useful defaults for the first remote task workflow.</p>
            </div>
          </div>
          <div className="space-y-3 text-sm text-slate-600">
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="font-medium text-slate-900">Exporter rollout preset</p>
              <p className="mt-1">
                The default preset is prefilled with
                {' '}
                <code className="text-xs">swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1</code>
                {' '}
                for one-click rollout.
              </p>
            </div>
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="font-medium text-slate-900">Remote script mode</p>
              <p className="mt-1">The backend downloads the script via curl or wget, then pipes it to bash on each robot.</p>
            </div>
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="font-medium text-slate-900">Per-robot results</p>
              <p className="mt-1">Every task stores command, output, error, and final status for each target robot.</p>
            </div>
          </div>
        </div>
      </div>

      <div className="bg-white border border-slate-200 rounded-2xl p-6">
        <div className="flex items-center justify-between gap-4 mb-4">
          <div>
            <h2 className="text-lg font-semibold text-slate-900">Task History</h2>
            <p className="text-sm text-slate-500">{tasks.length} dispatched task(s)</p>
          </div>
          {loading && <span className="text-sm text-slate-500">Loading...</span>}
        </div>

        <div className="space-y-3">
          {tasks.length === 0 ? (
            <div className="text-center py-12 border border-dashed border-slate-200 rounded-2xl">
              <Terminal className="w-12 h-12 text-slate-300 mx-auto mb-3" />
              <p className="text-slate-500">No remote tasks yet.</p>
            </div>
          ) : (
            tasks.map((task) => {
              const projectName = projects.find((project) => project.id === task.projectId)?.name;
              const isExpanded = expandedTaskId === task.id;
              return (
                <div key={task.id} className="border border-slate-200 rounded-2xl overflow-hidden">
                  <button
                    type="button"
                    onClick={() => setExpandedTaskId(isExpanded ? null : task.id)}
                    className="w-full px-5 py-4 flex items-start justify-between gap-4 hover:bg-slate-50 transition-colors"
                  >
                    <div className="text-left min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-medium text-slate-900">{task.name}</span>
                        <span className={`px-2.5 py-1 rounded-full border text-xs font-medium ${statusClassName(task.status)}`}>
                          {task.status}
                        </span>
                        <span className="px-2.5 py-1 rounded-full bg-slate-100 text-slate-600 text-xs">
                          {executionTypeLabel(task.executionType)}
                        </span>
                      </div>
                      <div className="flex flex-wrap gap-4 mt-2 text-sm text-slate-500">
                        <span>{scopeLabel(task.scope, projectName)}</span>
                        <span>{task.runs.length} robot(s)</span>
                        <span>{new Date(task.createdAt).toLocaleString()}</span>
                      </div>
                      {task.commandPreview && (
                        <code className="block mt-3 text-xs bg-slate-900 text-slate-100 px-3 py-2 rounded-xl whitespace-pre-wrap break-all">
                          {task.commandPreview}
                        </code>
                      )}
                    </div>
                    {isExpanded ? (
                      <ChevronDown className="w-5 h-5 text-slate-400 shrink-0 mt-1" />
                    ) : (
                      <ChevronRight className="w-5 h-5 text-slate-400 shrink-0 mt-1" />
                    )}
                  </button>

                  {isExpanded && (
                    <div className="border-t border-slate-200 px-5 py-4 space-y-3 bg-slate-50/60">
                      {task.runs.map((run) => (
                        <div key={`${task.id}-${run.serverId}`} className="bg-white border border-slate-200 rounded-2xl p-4 space-y-3">
                          <div className="flex flex-wrap items-center justify-between gap-3">
                            <div>
                              <div className="flex items-center gap-2 text-slate-900 font-medium">
                                <Server className="w-4 h-4 text-slate-500" />
                                {run.serverName || run.serverId}
                              </div>
                              <div className="text-sm text-slate-500 mt-1">
                                {run.startedAt ? new Date(run.startedAt).toLocaleString() : 'Pending'}
                                {run.finishedAt ? ` -> ${new Date(run.finishedAt).toLocaleString()}` : ''}
                              </div>
                            </div>
                            <span className={`px-2.5 py-1 rounded-full border text-xs font-medium ${statusClassName(run.status)}`}>
                              {run.status}
                            </span>
                          </div>

                          {run.error && (
                            <div className="flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
                              <span>{run.error}</span>
                            </div>
                          )}

                          {(run.output || run.command) && (
                            <div className="grid gap-3 lg:grid-cols-2">
                              <div>
                                <div className="text-xs font-medium text-slate-500 uppercase tracking-wide mb-2">Command</div>
                                <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto whitespace-pre-wrap break-all">
                                  {run.command || task.commandPreview}
                                </pre>
                              </div>
                              <div>
                                <div className="text-xs font-medium text-slate-500 uppercase tracking-wide mb-2">Output</div>
                                <pre className="text-xs bg-slate-900 text-slate-100 rounded-xl p-3 overflow-x-auto whitespace-pre-wrap break-all min-h-[88px]">
                                  {run.output || 'No output captured.'}
                                </pre>
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};
