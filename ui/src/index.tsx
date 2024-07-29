import ReactDOM from "react-dom/client";
import "./index.css";
import App from "./App";
import { ThemeProvider } from "@ui5/webcomponents-react";

import '@ui5/webcomponents-react/dist/Assets';

const root = ReactDOM.createRoot(
  document.getElementById("root") as HTMLElement
);
root.render(
    <ThemeProvider>
      <App />
    </ThemeProvider>
);