import type { Metadata } from "next";
export const metadata: Metadata = { title: "Leo — AI Eval Platform" };
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body style={{background:"#0a0c0f",minHeight:"100vh"}}>
        <nav style={{borderBottom:"1px solid #1e2530",padding:"0 24px",height:56,display:"flex",alignItems:"center"}}>
          <span style={{color:"white",fontWeight:700}}>leo</span>
        </nav>
        <main style={{maxWidth:1200,margin:"0 auto",padding:"32px 24px"}}>{children}</main>
      </body>
    </html>
  );
}
