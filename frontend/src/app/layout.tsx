import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "JSON API Dashboard",
  description: "Manage your JSON data with a simple API",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">
        {children}
      </body>
    </html>
  );
}
