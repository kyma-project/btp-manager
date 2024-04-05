import './App.css';
import * as ui5 from "@ui5/webcomponents-react";
import ServiceInstances from "./components/ServiceInstances"
import ServiceOfferings from "./components/ServiceOfferings"
import Secrets from "./components/Secrets"
import {useState} from "react";
import Overview from "./components/Overview";

function App() {
  return (
    <div className="App">
      <body className="ui5-content-density-compact">
        <Overview />
      </body>
    </div>
  );
}

export default App;