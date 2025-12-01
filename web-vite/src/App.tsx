import { RouterProvider, createBrowserRouter } from 'react-router-dom';
import { ToastProvider } from './components/ToastContainer';
import routes from './router';

function App() {
  const router = createBrowserRouter(routes);

  return (
    <ToastProvider>
      <RouterProvider router={router} />
    </ToastProvider>
  );
}

export default App;
