import type { Metadata } from "next";
import "./styles.css";

export const metadata: Metadata = {
  title: "Banco Costa",
  description: "Panel de visibilidad de transferencias de solo lectura de Banco Costa",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="es">
      <body>{children}</body>
    </html>
  );
}
