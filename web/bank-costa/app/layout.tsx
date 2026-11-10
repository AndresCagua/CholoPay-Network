import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  title: "Banco Costa",
  description: "Banco Costa read-only transfer visibility panel",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
