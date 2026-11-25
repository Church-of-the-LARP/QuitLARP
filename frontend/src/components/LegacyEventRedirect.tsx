import { Navigate, useParams } from 'react-router-dom';

// Legacy /events/<id> links predate the collaboration/interview split. They
// always pointed at a collaboration-style room, so keep that destination.
export default function LegacyEventRedirect() {
  const { id } = useParams();
  return <Navigate to={`/collaborations/${id}`} replace />;
}
