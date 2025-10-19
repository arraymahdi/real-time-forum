"use client";

import { useEffect } from "react";
import { useRouter } from "next/router";
import { MessageCircle, FileText, Users, User, Search } from "lucide-react";
import Link from "next/link";

export default function Footer() {
  const router = useRouter();

  useEffect(() => {
    const token = localStorage.getItem("token");
    if (!token) {
      router.push("/signin");
    }
  }, [router]);

  return (
    <footer className="fixed bottom-0 left-0 right-0 bg-white border-t border-gray-200 shadow-md flex justify-around py-2">
      <Link href="/chats">
        <MessageCircle className="w-6 h-6 text-gray-700 hover:text-blue-500" />
      </Link>
      <Link href="/posts">
        <FileText className="w-6 h-6 text-gray-700 hover:text-blue-500" />
      </Link>
      <Link href="/groups">
        <Users className="w-6 h-6 text-gray-700 hover:text-blue-500" />
      </Link>
      <Link href="/search">
        <Search className="w-6 h-6 text-gray-700 hover:text-blue-500" />
      </Link>
      <Link href="/profile">
        <User className="w-6 h-6 text-gray-700 hover:text-blue-500" />
      </Link>
    </footer>
  );
}
