import { useState, useEffect } from 'react';
import { useAuth } from '../context/AuthContext';
import api from '../services/api';
import {
  FolderIcon,
  DocumentIcon,
  UsersIcon,
  ShareIcon,
} from '@heroicons/react/24/outline';

const Dashboard = () => {
  const { user } = useAuth();
  const [stats, setStats] = useState({
    folders: 0,
    notes: 0,
    sharedItems: 0,
    teams: 0,
  });
  const [recentActivity, setRecentActivity] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchDashboardData = async () => {
      try {
        setLoading(true);
        
        // Fetch folders
        const foldersResponse = await api.get('/folders');
        const folders = foldersResponse.data;

        // Fetch notes
        const notesResponse = await api.get('/notes');
        const notes = notesResponse.data;

        // Fetch shared items
        const sharedResponse = await api.get('/shares');
        const shared = sharedResponse.data;

        setStats({
          folders: folders.length,
          notes: notes.length,
          sharedItems: shared.length,
          teams: 0, // Temporarily set to 0 since we need userId for teams query
        });

        // Create recent activity from notes and folders
        const activities = [
          ...folders.slice(0, 3).map(folder => ({
            id: folder.id,
            type: 'folder',
            title: folder.title,
            action: 'created',
            date: new Date(folder.created_at),
          })),
          ...notes.slice(0, 3).map(note => ({
            id: note.id,
            type: 'note',
            title: note.title,
            action: 'created',
            date: new Date(note.created_at),
          })),
        ];

        activities.sort((a, b) => b.date - a.date);
        setRecentActivity(activities.slice(0, 5));
      } catch (error) {
        console.error('Error fetching dashboard data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchDashboardData();
  }, []);

  const StatCard = ({ title, value, icon: Icon, color }) => (
    <div className="bg-white overflow-hidden shadow rounded-lg">
      <div className="p-5">
        <div className="flex items-center">
          <div className="flex-shrink-0">
            <Icon className={`h-6 w-6 ${color}`} />
          </div>
          <div className="ml-5 w-0 flex-1">
            <dl>
              <dt className="text-sm font-medium text-gray-500 truncate">
                {title}
              </dt>
              <dd className="text-lg font-medium text-gray-900">{value}</dd>
            </dl>
          </div>
        </div>
      </div>
    </div>
  );

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">
          Welcome back, {user?.username}!
        </h1>
        <p className="mt-1 text-sm text-gray-600">
          Here's what's happening with your training assets.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 mb-8">
        <StatCard
          title="Teams"
          value={stats.teams}
          icon={UsersIcon}
          color="text-blue-500"
        />
        <StatCard
          title="Folders"
          value={stats.folders}
          icon={FolderIcon}
          color="text-yellow-500"
        />
        <StatCard
          title="Notes"
          value={stats.notes}
          icon={DocumentIcon}
          color="text-green-500"
        />
        <StatCard
          title="Shared Items"
          value={stats.sharedItems}
          icon={ShareIcon}
          color="text-purple-500"
        />
      </div>

      {/* Recent Activity */}
      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
            Recent Activity
          </h3>
          {recentActivity.length > 0 ? (
            <div className="flow-root">
              <ul className="-mb-8">
                {recentActivity.map((activity, index) => (
                  <li key={activity.id}>
                    <div className="relative pb-8">
                      {index !== recentActivity.length - 1 && (
                        <span
                          className="absolute top-4 left-4 -ml-px h-full w-0.5 bg-gray-200"
                          aria-hidden="true"
                        />
                      )}
                      <div className="relative flex space-x-3">
                        <div>
                          <span className="h-8 w-8 rounded-full bg-gray-400 flex items-center justify-center ring-8 ring-white">
                            {activity.type === 'folder' ? (
                              <FolderIcon className="h-4 w-4 text-white" />
                            ) : (
                              <DocumentIcon className="h-4 w-4 text-white" />
                            )}
                          </span>
                        </div>
                        <div className="min-w-0 flex-1 pt-1.5 flex justify-between space-x-4">
                          <div>
                            <p className="text-sm text-gray-500">
                              {activity.action === 'created' ? 'Created' : 'Updated'}{' '}
                              <span className="font-medium text-gray-900">
                                {activity.title}
                              </span>
                            </p>
                          </div>
                          <div className="text-right text-sm whitespace-nowrap text-gray-500">
                            {activity.date.toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <p className="text-gray-500 text-center py-4">
              No recent activity to show.
            </p>
          )}
        </div>
      </div>
    </div>
  );
};

export default Dashboard;
