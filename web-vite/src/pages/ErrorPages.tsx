import { Link, isRouteErrorResponse, useRouteError } from "react-router-dom";
import { AlertTriangle, Compass } from "lucide-react";
import { EmptyState } from "@/components/ui";

export function RouteErrorPage() {
  const error = useRouteError();
  const message = isRouteErrorResponse(error)
    ? `${error.status} ${error.statusText}`
    : error instanceof Error
      ? error.message
      : "The page failed to load.";

  return (
    <div className="flex min-h-screen items-center justify-center bg-canvas p-6">
      <div className="card w-full max-w-md">
        <EmptyState
          icon={<AlertTriangle size={20} />}
          title="Something went wrong"
          description={
            <>
              {message}
              <br />
              If JVP was just upgraded, reloading usually fixes this.
            </>
          }
          action={
            <button className="btn-primary" onClick={() => window.location.reload()}>
              Reload page
            </button>
          }
        />
      </div>
    </div>
  );
}

export function NotFoundPage() {
  return (
    <div className="card">
      <EmptyState
        icon={<Compass size={20} />}
        title="Page not found"
        description="The page you are looking for does not exist or has been moved."
        action={
          <Link to="/" className="btn-primary">
            Back to overview
          </Link>
        }
      />
    </div>
  );
}
