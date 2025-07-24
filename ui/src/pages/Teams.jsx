import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import {
  UsersIcon,
} from '@heroicons/react/24/outline';

const Teams = () => {
  const { isManager } = useAuth();

  return (
    <div>
      <div className="sm:flex sm:items-center mb-8">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-bold text-gray-900">Teams</h1>
          <p className="mt-2 text-sm text-gray-700">
            Team management functionality will be available soon.
          </p>
        </div>
      </div>

      {/* Teams List */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <div className="px-6 py-8 text-center">
          <UsersIcon className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">Teams Feature Coming Soon</h3>
          <p className="mt-1 text-sm text-gray-500">
            {isManager
              ? 'Team management features are being developed.'
              : 'You will be able to view your teams here.'}
          </p>
        </div>
      </div>
    </div>
  );
};

export default Teams;
