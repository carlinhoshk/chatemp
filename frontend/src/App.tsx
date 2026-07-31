import { Navigate, Route, Routes } from "react-router-dom";
import HomeScreen from "./components/HomeScreen";
import ChatRoom from "./components/ChatRoom";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<HomeScreen />} />
      <Route path="/room/:code" element={<ChatRoom />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
