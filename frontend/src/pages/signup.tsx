"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";

export default function Signup() {
  const router = useRouter();
  const [form, setForm] = useState({
    email: "",
    password: "",
    first_name: "",
    last_name: "",
    date_of_birth: "",
    nickname: "",
    about_me: "",
    profile_type: "public",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const handleChange = (
    e: React.ChangeEvent<
      HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
    >
  ) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");
    setSuccess("");

    try {
      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/register`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(form),
        }
      );

      if (!res.ok) {
        const err = await res.text();
        throw new Error(err || "Signup failed");
      }

      setSuccess("Signup successful! Redirecting to sign in...");
      setTimeout(() => router.push("/signin"), 2500);
    } catch (err: any) {
      setError(err.message || "Signup failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-lg mx-auto bg-white shadow-lg rounded-lg p-8">
      <h1 className="text-3xl font-bold mb-6 text-center text-black">
        Create an Account
      </h1>
      <form onSubmit={handleSubmit} className="space-y-4">
        <input
          name="email"
          type="email"
          placeholder="Email *"
          required
          value={form.email}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 placeholder-gray-400 text-gray-900"
        />

        <input
          name="password"
          type="password"
          placeholder="Password *"
          required
          value={form.password}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 placeholder-gray-400 text-gray-900"
        />

        <input
          name="nickname"
          type="text"
          placeholder="Nickname *"
          required
          value={form.nickname}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 placeholder-gray-400 text-gray-900"
        />

        <select
          name="profile_type"
          value={form.profile_type}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 text-gray-900"
        >
          <option value="public">Public Profile</option>
          <option value="private">Private Profile</option>
        </select>

        <input
          name="first_name"
          type="text"
          placeholder="First Name"
          value={form.first_name}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 placeholder-gray-400 text-gray-900"
        />

        <input
          name="last_name"
          type="text"
          placeholder="Last Name"
          value={form.last_name}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 placeholder-gray-400 text-gray-900"
        />

        <input
          name="date_of_birth"
          type="date"
          value={form.date_of_birth}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 text-gray-900"
        />

        <textarea
          name="about_me"
          placeholder="About Me"
          value={form.about_me}
          onChange={handleChange}
          className="w-full border p-3 rounded focus:ring focus:ring-blue-200 h-24 resize-none placeholder-gray-400 text-gray-900"
        />

        {error && <p className="text-red-500 text-sm">{error}</p>}
        {success && <p className="text-green-600 text-sm">{success}</p>}

        <button
          type="submit"
          disabled={loading}
          className="w-full bg-blue-600 text-white font-semibold py-3 rounded hover:bg-blue-700 transition disabled:opacity-50"
        >
          {loading ? "Creating Account..." : "Sign Up"}
        </button>

        <p className="text-center text-gray-600 text-sm">
          Already have an account?{" "}
          <Link href="/signin" className="text-blue-600 hover:underline">
            Sign In
          </Link>
        </p>
      </form>
    </div>
  );
}
