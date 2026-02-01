import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { MantineProvider } from "@mantine/core";
import "@mantine/core/styles.css";
import "./main.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <MantineProvider>
      <h1>1mail</h1>
    </MantineProvider>
  </StrictMode>,
);
