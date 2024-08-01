import './App.css';
import ServiceInstancesView from './components/ServiceInstancesView';
import ServiceOfferingsView from './components/ServiceOfferingsView';

import * as React from "react";
import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import Layout from './components/Layout';

function App() {
  
  const [secret, setSecret] = React.useState("");

  return (
    <div id="App" className="App html-wrap flex-container flex-column">
        <BrowserRouter basename='/' >
          <Routes>
            <Route path="/" element={<Layout onSecretChanged={(s: string) => setSecret(s)}/>}>
              <Route index element={<Navigate to="offerings" replace />} />
              <Route path="*" element={<Navigate to="offerings" replace />} />

              <Route path="/instances" element={<ServiceInstancesView/>}/>
              <Route path="/offerings" element={<ServiceOfferingsView secret={secret}/>}/>
            </Route>
          </Routes>
        </BrowserRouter>
    </div>
  );
}

export default App;