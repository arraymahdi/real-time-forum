"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link"; // ✅ import Link
import { useAuth } from "../context/AuthContext";

export default function Signin() {
  const router = useRouter();
  const { login } = useAuth();
  const [form, setForm] = useState({ email: "", password: "" });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/login`, // ✅ fixed template literal
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(form),
        }
      );

      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || "Login failed");
      }

      const data = await res.json();

      if (data.token) {
        login(data.token);
        router.push("/chats");
      } else {
        throw new Error("No token received from server");
      }
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-md mx-auto bg-white shadow-lg rounded-lg p-8">
      <h1 className="text-3xl font-bold mb-6 text-center text-black">Sign In</h1>
      <form onSubmit={handleSubmit} className="space-y-4">
        <input
          name="email"
          type="email"
          placeholder="Email"
          value={form.email}
          onChange={handleChange}
          required
          className="w-full border p-3 rounded focus:ring focus:ring-green-200 placeholder-gray-300 text-gray-900"
        />
        <input
          name="password"
          type="password"
          placeholder="Password"
          value={form.password}
          onChange={handleChange}
          required
          className="w-full border p-3 rounded focus:ring focus:ring-green-200 placeholder-gray-300 text-gray-900"
        />

        {error && <p className="text-red-500 text-sm">{error}</p>}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-green-600 text-white font-semibold py-3 rounded hover:bg-green-700 transition disabled:opacity-50"
        >
          {loading ? "Signing In..." : "Sign In"}
        </button>
      </form>

      {/* ✅ Signup link */}
      <p className="mt-4 text-center text-gray-600 text-sm">
        Don’t have an account?{" "}
        <Link href="/signup" className="text-green-600 font-medium hover:underline">
          Sign Up
        </Link>
      </p>
    </div>
  );
}
