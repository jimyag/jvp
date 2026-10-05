import { RouterProvider, createBrowserRouter } from "react-router-dom";
import { ToastProvider } from "./components/ToastContainer";
import routes from "./router";

const router = createBrowserRouter(routes);

function App() {
  return (
    <ToastProvider>
      <RouterProvider router={router} />
    </ToastProvider>
  );
}

export default App;
