"use server";

export default async function Home() {
  return (
    <div className="container">
      <p>Join a new session or continue with an existing one.</p>
      <a href="/timer">Start new timer session</a>
    </div>
  );
}
