"use client";

import { ReactNode } from "react";
import { useRouter } from "next/router";
import Navbar from "./Navbar";
import Footer from "./Footer";

export default function Layout({ children }: { children: ReactNode }) {
  const router = useRouter();

  // Pages where we don't want Navbar/Footer
  const authPages = ["/signin", "/signup"];
  const isAuthPage = authPages.includes(router.pathname);

  return (
    <div className="min-h-screen flex flex-col">
      {!isAuthPage && <Navbar />}
      <main className="flex-1 p-6">{children}</main>
      {!isAuthPage && <Footer />}
    </div>
  );
}
