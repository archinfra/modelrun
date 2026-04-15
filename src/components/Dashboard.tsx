import { motion } from 'framer-motion';
import {
  Rocket,
  Server,
  Database,
  Activity,
  Plus,
  ChevronRight,
  CheckCircle2,
  XCircle,
  Clock,
  Play,
  FolderKanban,
  Cpu,
  HardDrive,
  Thermometer,
} from 'lucide-react';
import { useAppStore } from '../store';
import { Link } from 'react-router-dom';

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.1 },
  },
};

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: { opacity: 1, y: 0 },
};

export default function Dashboard() {
  const { projects, currentProjectId, servers, deployments } = useAppStore();

  const currentProject = projects.find((p) => p.id === currentProjectId);
  const projectServers = servers.filter((s) => s.projectId === currentProjectId);
  const projectDeployments = deployments.filter((d) =>
    projectServers.some((s) => d.servers.includes(s.id))
  );

  const stats = [
    {
      label: '运行中',
      value: projectDeployments.filter((d) => d.status === 'running').length,
      icon: Play,
      color: 'text-emerald-500',
      bg: 'bg-emerald-50',
    },
    {
      label: '总部署',
      value: projectDeployments.length,
      icon: Rocket,
      color: 'text-blue-500',
      bg: 'bg-blue-50',
    },
    {
      label: '服务器',
      value: projectServers.length,
      icon: Server,
      color: 'text-purple-500',
      bg: 'bg-purple-50',
    },
    {
      label: 'GPU 总数',
      value: projectServers.reduce((acc, s) => acc + (s.gpuInfo?.length || 0), 0),
      icon: Cpu,
      color: 'text-orange-500',
      bg: 'bg-orange-50',
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">
            {currentProject?.name || '项目看板'}
          </h1>
          <p className="text-slate-500 mt-1">管理和监控您的 AI 模型部署</p>
        </div>
        <Link
          to="/wizard"
          className="flex items-center gap-2 px-5 py-2.5 bg-blue-600 hover:bg-blue-700 text-white rounded-xl font-medium transition-colors shadow-lg shadow-blue-500/25"
        >
          <Plus className="w-5 h-5" />
          新建部署
        </Link>
      </div>

      {/* Stats Grid */}
      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="grid grid-cols-4 gap-4"
      >
        {stats.map((stat) => (
          <motion.div
            key={stat.label}
            variants={itemVariants}
            className={`${stat.bg} rounded-2xl p-5 border border-slate-100`}
          >
            <div className={`w-12 h-12 ${stat.bg} rounded-xl flex items-center justify-center mb-4`}>
              <stat.icon className={`w-6 h-6 ${stat.color}`} />
            </div>
            <div className="text-3xl font-bold text-slate-900">{stat.value}</div>
            <div className="text-sm text-slate-500 mt-1">{stat.label}</div>
          </motion.div>
        ))}
      </motion.div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-3 gap-6">
        {/* Server Status */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
          className="col-span-2 bg-white rounded-2xl border border-slate-200 p-6"
        >
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-semibold text-slate-900 flex items-center gap-2">
              <Server className="w-5 h-5 text-slate-400" />
              服务器状态
            </h2>
            <Link
              to="/servers"
              className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
            >
              查看全部
              <ChevronRight className="w-4 h-4" />
            </Link>
          </div>

          <div className="space-y-3">
            {projectServers.length === 0 ? (
              <div className="text-center py-12 text-slate-400">
                <Server className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>暂无服务器</p>
                <p className="text-sm mt-1">请先添加服务器到项目</p>
              </div>
            ) : (
              projectServers.slice(0, 4).map((server) => (
                <div
                  key={server.id}
                  className="flex items-center justify-between p-4 bg-slate-50 rounded-xl hover:bg-slate-100 transition-colors"
                >
                  <div className="flex items-center gap-4">
                    <div
                      className={`w-3 h-3 rounded-full ${
                        server.status === 'online' ? 'bg-emerald-500' : 'bg-red-500'
                      }`}
                    />
                    <div>
                      <h3 className="font-medium text-slate-900">{server.name}</h3>
                      <p className="text-sm text-slate-500">
                        {server.host} · {server.gpuInfo?.length || 0} GPU
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-6 text-sm">
                    {server.gpuInfo?.map((gpu, idx) => (
                      <div key={idx} className="flex items-center gap-2">
                        <Thermometer className="w-4 h-4 text-orange-400" />
                        <span className="text-slate-600">{gpu.temperature}°C</span>
                        <div className="w-20 h-1.5 bg-slate-200 rounded-full">
                          <div
                            className="h-full bg-blue-500 rounded-full"
                            style={{ width: `${gpu.utilization}%` }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        </motion.div>

        {/* Recent Deployments */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
          className="bg-white rounded-2xl border border-slate-200 p-6"
        >
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-semibold text-slate-900 flex items-center gap-2">
              <Activity className="w-5 h-5 text-slate-400" />
              最近部署
            </h2>
            <Link
              to="/deployments"
              className="text-sm text-blue-600 hover:text-blue-700 flex items-center gap-1"
            >
              查看全部
              <ChevronRight className="w-4 h-4" />
            </Link>
          </div>

          <div className="space-y-3">
            {projectDeployments.length === 0 ? (
              <div className="text-center py-12 text-slate-400">
                <Rocket className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>暂无部署</p>
                <p className="text-sm mt-1">点击上方按钮创建</p>
              </div>
            ) : (
              projectDeployments.slice(0, 5).map((deployment) => (
                <div
                  key={deployment.id}
                  className="flex items-center justify-between p-3 hover:bg-slate-50 rounded-xl transition-colors"
                >
                  <div className="flex items-center gap-3">
                    {deployment.status === 'running' && (
                      <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                    )}
                    {deployment.status === 'failed' && (
                      <XCircle className="w-5 h-5 text-red-500" />
                    )}
                    {deployment.status === 'deploying' && (
                      <Clock className="w-5 h-5 text-blue-500" />
                    )}
                    <div>
                      <h3 className="font-medium text-slate-900">{deployment.name}</h3>
                      <p className="text-sm text-slate-500">{deployment.model.name}</p>
                    </div>
                  </div>
                  <span
                    className={`px-2 py-1 rounded-lg text-xs font-medium ${
                      deployment.status === 'running'
                        ? 'bg-emerald-100 text-emerald-700'
                        : deployment.status === 'failed'
                        ? 'bg-red-100 text-red-700'
                        : 'bg-blue-100 text-blue-700'
                    }`}
                  >
                    {deployment.status}
                  </span>
                </div>
              ))
            )}
          </div>
        </motion.div>
      </div>

      {/* Quick Actions */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.4 }}
        className="grid grid-cols-3 gap-4"
      >
        {[
          { label: '添加服务器', icon: Server, path: '/servers', color: 'from-blue-500 to-blue-600' },
          { label: '导入模型', icon: Database, path: '/models', color: 'from-purple-500 to-purple-600' },
          { label: '查看文档', icon: FolderKanban, path: '#', color: 'from-orange-500 to-orange-600' },
        ].map((action) => (
          <Link
            key={action.label}
            to={action.path}
            className={`flex items-center gap-4 p-5 bg-gradient-to-r ${action.color} rounded-2xl text-white hover:shadow-lg transition-all`}
          >
            <action.icon className="w-8 h-8" />
            <div>
              <div className="font-semibold">{action.label}</div>
              <ChevronRight className="w-4 h-4 mt-1" />
            </div>
          </Link>
        ))}
      </motion.div>
    </div>
  );
}
