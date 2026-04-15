import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import {
  Project,
  ServerConfig,
  JumpHost,
  ModelConfig,
  DeploymentConfig,
  DeploymentTask,
  WizardState,
  WizardStep,
} from '../types';

interface AppState {
  projects: Project[];
  servers: ServerConfig[];
  jumpHosts: JumpHost[];
  models: ModelConfig[];
  deployments: DeploymentConfig[];
  tasks: DeploymentTask[];
  wizard: WizardState;
  currentProjectId: string | null;

  addProject: (project: Project) => void;
  updateProject: (id: string, project: Partial<Project>) => void;
  removeProject: (id: string) => void;
  setCurrentProject: (id: string | null) => void;

  addServer: (server: ServerConfig) => void;
  updateServer: (id: string, server: Partial<ServerConfig>) => void;
  removeServer: (id: string) => void;

  addJumpHost: (host: JumpHost) => void;
  updateJumpHost: (id: string, host: Partial<JumpHost>) => void;
  removeJumpHost: (id: string) => void;

  addModel: (model: ModelConfig) => void;
  updateModel: (id: string, model: Partial<ModelConfig>) => void;
  removeModel: (id: string) => void;

  addDeployment: (deployment: DeploymentConfig) => void;
  updateDeployment: (id: string, deployment: Partial<DeploymentConfig>) => void;
  removeDeployment: (id: string) => void;

  addTask: (task: DeploymentTask) => void;
  updateTask: (id: string, task: Partial<DeploymentTask>) => void;
  removeTask: (id: string) => void;

  setWizardStep: (step: WizardStep) => void;
  completeWizardStep: (step: WizardStep) => void;
  updateWizardConfig: (config: Partial<DeploymentConfig>) => void;
  resetWizard: () => void;
}

const initialWizardState: WizardState = {
  currentStep: 'model',
  completedSteps: [],
  config: {},
};

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      projects: [],
      servers: [],
      jumpHosts: [],
      models: [],
      deployments: [],
      tasks: [],
      wizard: initialWizardState,
      currentProjectId: null,

      addProject: (project) =>
        set((state) => ({ projects: [...state.projects, project] })),
      updateProject: (id, project) =>
        set((state) => ({
          projects: state.projects.map((p) =>
            p.id === id ? { ...p, ...project } : p
          ),
        })),
      removeProject: (id) =>
        set((state) => ({
          projects: state.projects.filter((p) => p.id !== id),
          servers: state.servers.filter((s) => s.projectId !== id),
        })),
      setCurrentProject: (id) =>
        set(() => ({ currentProjectId: id })),

      addServer: (server) =>
        set((state) => ({ servers: [...state.servers, server] })),
      updateServer: (id, server) =>
        set((state) => ({
          servers: state.servers.map((s) =>
            s.id === id ? { ...s, ...server } : s
          ),
        })),
      removeServer: (id) =>
        set((state) => ({
          servers: state.servers.filter((s) => s.id !== id),
        })),

      addJumpHost: (host) =>
        set((state) => ({ jumpHosts: [...state.jumpHosts, host] })),
      updateJumpHost: (id, host) =>
        set((state) => ({
          jumpHosts: state.jumpHosts.map((h) =>
            h.id === id ? { ...h, ...host } : h
          ),
        })),
      removeJumpHost: (id) =>
        set((state) => ({
          jumpHosts: state.jumpHosts.filter((h) => h.id !== id),
        })),

      addModel: (model) =>
        set((state) => ({ models: [...state.models, model] })),
      updateModel: (id, model) =>
        set((state) => ({
          models: state.models.map((m) =>
            m.id === id ? { ...m, ...model } : m
          ),
        })),
      removeModel: (id) =>
        set((state) => ({
          models: state.models.filter((m) => m.id !== id),
        })),

      addDeployment: (deployment) =>
        set((state) => ({ deployments: [...state.deployments, deployment] })),
      updateDeployment: (id, deployment) =>
        set((state) => ({
          deployments: state.deployments.map((d) =>
            d.id === id ? { ...d, ...deployment } : d
          ),
        })),
      removeDeployment: (id) =>
        set((state) => ({
          deployments: state.deployments.filter((d) => d.id !== id),
        })),

      addTask: (task) =>
        set((state) => ({ tasks: [...state.tasks, task] })),
      updateTask: (id, task) =>
        set((state) => ({
          tasks: state.tasks.map((t) =>
            t.id === id ? { ...t, ...task } : t
          ),
        })),
      removeTask: (id) =>
        set((state) => ({
          tasks: state.tasks.filter((t) => t.id !== id),
        })),

      setWizardStep: (step) =>
        set((state) => ({
          wizard: { ...state.wizard, currentStep: step },
        })),
      completeWizardStep: (step) =>
        set((state) => ({
          wizard: {
            ...state.wizard,
            completedSteps: [...state.wizard.completedSteps, step],
          },
        })),
      updateWizardConfig: (config) =>
        set((state) => ({
          wizard: {
            ...state.wizard,
            config: { ...state.wizard.config, ...config },
          },
        })),
      resetWizard: () =>
        set(() => ({
          wizard: initialWizardState,
        })),
    }),
    {
      name: 'model-deploy-storage',
    }
  )
);
