import './App.css';
import ServiceInstancesView from './components/ServiceInstancesView';
import ServiceOfferingsView from './components/ServiceOfferingsView';

import * as React from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import Layout from './components/Layout';

function App() {

  const [secret, setSecret] = React.useState("");
  const [title, setTitle] = React.useState("");

  return (
    <div id="App" className="App html-wrap flex-container flex-column">
      <BrowserRouter basename='/' >
        <Routes>
          <Route path="/" element={<Layout title={title} onSecretChanged={(s: string) => setSecret(s)}  /> }>
            <Route index element={<Navigate to="offerings" replace />} />
            <Route path="*" element={<Navigate to="offerings" replace />} />

            <Route path="/instances" element={<ServiceInstancesView setTitle={(title: string) => setTitle(title)} secret={secret} />} />
            <Route path="/instances/:id" element={<ServiceInstancesView setTitle={(title: string) => setTitle(title)} secret={secret} />} />
            <Route path="/offerings" element={<ServiceOfferingsView setTitle={(title: string) => setTitle(title)} secret={secret} />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </div>
  );
}

export default App;