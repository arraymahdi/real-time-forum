import "../styles/globals.css";
import type { AppProps } from "next/app";
import { AuthProvider } from "../context/AuthContext";
import { WebSocketProvider } from "../context/WebSocketContext";
import Layout from "../components/Layout";

export default function App({ Component, pageProps }: AppProps) {
  return (
    <AuthProvider>
      <WebSocketProvider>
          <Layout>
            <Component {...pageProps} />
          </Layout>
      </WebSocketProvider>
    </AuthProvider>
  );
}
