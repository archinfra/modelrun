import React from 'react';
import { HashRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import Dashboard from './components/Dashboard';
import { ServerManager } from './components/ServerManager';
import { DeploymentList } from './components/DeploymentList';
import { DeployWizard } from './components/DeployWizard';
import { ModelManager } from './components/ModelManager';

const App: React.FC = () => {
  return (
    <HashRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/models" element={<ModelManager />} />
          <Route path="/servers" element={<ServerManager />} />
          <Route path="/wizard" element={<DeployWizard />} />
          <Route path="/deployments" element={<DeploymentList />} />
        </Routes>
      </Layout>
    </HashRouter>
  );
};

export default App;
