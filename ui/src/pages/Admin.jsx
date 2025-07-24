import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { 
  UserGroupIcon,
} from '@heroicons/react/24/outline';

const Admin = () => {
  const { user } = useAuth();

  return (
    <div>
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-gray-900">Admin Panel</h1>
        <p className="mt-2 text-sm text-gray-700">
          Administrative features are being developed.
        </p>
      </div>

      {/* Admin Features Placeholder */}
      <div className="bg-white shadow overflow-hidden sm:rounded-lg">
        <div className="px-6 py-8 text-center">
          <UserGroupIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">Admin Features Coming Soon</h3>
          <p className="mt-1 text-sm text-gray-500">
            User management and system administration features are being developed.
          </p>
          <div className="mt-4 text-xs text-gray-400">
            Logged in as: {user?.username} ({user?.role})
          </div>
        </div>
      </div>
    </div>
  );
};

export default Admin;
