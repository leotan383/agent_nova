import { Navigate, Route, Routes } from "react-router-dom";
import LibraryPage from "./pages/LibraryPage";
import StudioPage from "./pages/StudioPage";

export default function App() {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <Routes>
        <Route path="/" element={<LibraryPage />} />
        <Route path="/studio" element={<StudioPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </div>
  );
}
