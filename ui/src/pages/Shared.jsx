import { useState, useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { GET_USERS_QUERY } from '../services/graphql';
import api from '../services/api';
import {
  ShareIcon,
  FolderIcon,
  DocumentIcon,
  EyeIcon,
  PencilIcon,
  UserIcon,
  XMarkIcon,
  PlusIcon,
} from '@heroicons/react/24/outline';

const Shared = () => {
  const [shares, setShares] = useState([]);
  const [folders, setFolders] = useState([]);
  const [notes, setNotes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateShare, setShowCreateShare] = useState(false);
  const [newShareData, setNewShareData] = useState({
    user_id: '',
    resource_type: 'folder',
    resource_id: '',
    access_level: 'read',
  });

  const { data: usersData } = useQuery(GET_USERS_QUERY);

  useEffect(() => {
    fetchData();
  }, []);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [sharesResponse, foldersResponse, notesResponse] = await Promise.all([
        api.get('/shares'),
        api.get('/folders'),
        api.get('/notes'),
      ]);
      setShares(sharesResponse.data);
      setFolders(foldersResponse.data);
      setNotes(notesResponse.data);
    } catch (error) {
      console.error('Error fetching data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateShare = async (e) => {
    e.preventDefault();
    if (!newShareData.user_id || !newShareData.resource_id) return;

    try {
      await api.post('/shares', {
        user_id: parseInt(newShareData.user_id),
        resource_type: newShareData.resource_type,
        resource_id: parseInt(newShareData.resource_id),
        access_level: newShareData.access_level,
      });
      setNewShareData({
        user_id: '',
        resource_type: 'folder',
        resource_id: '',
        access_level: 'read',
      });
      setShowCreateShare(false);
      fetchData();
    } catch (error) {
      console.error('Error creating share:', error);
      alert('Failed to create share');
    }
  };

  const handleDeleteShare = async (shareId) => {
    if (!confirm('Are you sure you want to remove this share?')) {
      return;
    }

    try {
      await api.delete(`/shares/${shareId}`);
      fetchData();
    } catch (error) {
      console.error('Error deleting share:', error);
      alert('Failed to delete share');
    }
  };

  const handleUpdateShareAccess = async (shareId, newAccessLevel) => {
    try {
      await api.put(`/shares/${shareId}`, { access_level: newAccessLevel });
      fetchData();
    } catch (error) {
      console.error('Error updating share:', error);
      alert('Failed to update share access');
    }
  };

  const getResourceTitle = (share) => {
    if (share.resource_type === 'folder') {
      const folder = folders.find(f => f.id === share.resource_id);
      return folder ? folder.title : 'Unknown Folder';
    } else {
      const note = notes.find(n => n.id === share.resource_id);
      return note ? note.title : 'Unknown Note';
    }
  };

  const getResourceIcon = (resourceType) => {
    return resourceType === 'folder' ? FolderIcon : DocumentIcon;
  };

  const getAccessIcon = (accessLevel) => {
    return accessLevel === 'read' ? EyeIcon : PencilIcon;
  };

  const users = usersData?.getUsers || [];
  const availableResources = newShareData.resource_type === 'folder' ? folders : notes;

  if (loading) {
    return (
      <div className="flex justify-center items-center h-64">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-indigo-500"></div>
      </div>
    );
  }

  return (
    <div>
      <div className="sm:flex sm:items-center mb-8">
        <div className="sm:flex-auto">
          <h1 className="text-2xl font-bold text-gray-900">Shared Resources</h1>
          <p className="mt-2 text-sm text-gray-700">
            Manage shared folders and notes with team members.
          </p>
        </div>
        <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none">
          <button
            type="button"
            onClick={() => setShowCreateShare(true)}
            className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700"
          >
            <PlusIcon className="-ml-1 mr-2 h-4 w-4" />
            Share Resource
          </button>
        </div>
      </div>

      {/* Create Share Form */}
      {showCreateShare && (
        <div className="mb-8 bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Share Resource</h3>
          <form onSubmit={handleCreateShare} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Share with User
                </label>
                <select
                  value={newShareData.user_id}
                  onChange={(e) => setNewShareData({ ...newShareData, user_id: e.target.value })}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                  required
                >
                  <option value="">Select a user</option>
                  {users.map(user => (
                    <option key={user.id} value={user.id}>
                      {user.username} ({user.email})
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Resource Type
                </label>
                <select
                  value={newShareData.resource_type}
                  onChange={(e) => setNewShareData({ 
                    ...newShareData, 
                    resource_type: e.target.value,
                    resource_id: '' // Reset resource_id when type changes
                  })}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                >
                  <option value="folder">Folder</option>
                  <option value="note">Note</option>
                </select>
              </div>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  {newShareData.resource_type === 'folder' ? 'Folder' : 'Note'}
                </label>
                <select
                  value={newShareData.resource_id}
                  onChange={(e) => setNewShareData({ ...newShareData, resource_id: e.target.value })}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                  required
                >
                  <option value="">Select a {newShareData.resource_type}</option>
                  {availableResources.map(resource => (
                    <option key={resource.id} value={resource.id}>
                      {resource.title}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Access Level
                </label>
                <select
                  value={newShareData.access_level}
                  onChange={(e) => setNewShareData({ ...newShareData, access_level: e.target.value })}
                  className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                >
                  <option value="read">Read Only</option>
                  <option value="write">Read & Write</option>
                </select>
              </div>
            </div>
            <div className="flex gap-4 pt-4">
              <button
                type="submit"
                className="bg-indigo-600 text-white px-4 py-2 rounded-md hover:bg-indigo-700"
              >
                Share Resource
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowCreateShare(false);
                  setNewShareData({
                    user_id: '',
                    resource_type: 'folder',
                    resource_id: '',
                    access_level: 'read',
                  });
                }}
                className="bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Shares List */}
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <ul className="divide-y divide-gray-200">
          {shares.length > 0 ? (
            shares.map((share) => {
              const ResourceIcon = getResourceIcon(share.resource_type);
              const AccessIcon = getAccessIcon(share.access_level);
              const user = users.find(u => u.id === share.user_id);
              
              return (
                <li key={share.id} className="px-6 py-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center min-w-0 flex-1">
                      <div className="flex-shrink-0 mr-4">
                        <div className="relative">
                          <ResourceIcon className={`h-8 w-8 ${
                            share.resource_type === 'folder' ? 'text-yellow-500' : 'text-blue-500'
                          }`} />
                          <AccessIcon className={`h-4 w-4 absolute -bottom-1 -right-1 bg-white rounded-full p-0.5 ${
                            share.access_level === 'read' ? 'text-gray-500' : 'text-green-500'
                          }`} />
                        </div>
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center space-x-3">
                          <h3 className="text-lg font-medium text-gray-900 truncate">
                            {getResourceTitle(share)}
                          </h3>
                          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                            share.resource_type === 'folder' 
                              ? 'bg-yellow-100 text-yellow-800'
                              : 'bg-blue-100 text-blue-800'
                          }`}>
                            {share.resource_type}
                          </span>
                        </div>
                        <div className="mt-1 flex items-center space-x-4 text-sm text-gray-500">
                          <div className="flex items-center">
                            <UserIcon className="h-4 w-4 mr-1" />
                            <span>
                              Shared with: <span className="font-medium">
                                {user ? `${user.username} (${user.email})` : 'Unknown User'}
                              </span>
                            </span>
                          </div>
                          <div className="flex items-center">
                            <AccessIcon className="h-4 w-4 mr-1" />
                            <span className="capitalize">{share.access_level} access</span>
                          </div>
                        </div>
                        <p className="mt-1 text-xs text-gray-400">
                          Shared on: {new Date(share.created_at).toLocaleDateString()}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2 ml-4">
                      <select
                        value={share.access_level}
                        onChange={(e) => handleUpdateShareAccess(share.id, e.target.value)}
                        className="text-sm border border-gray-300 rounded-md px-2 py-1 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                      >
                        <option value="read">Read Only</option>
                        <option value="write">Read & Write</option>
                      </select>
                      <button
                        onClick={() => handleDeleteShare(share.id)}
                        className="text-red-600 hover:text-red-900 p-1"
                        title="Remove share"
                      >
                        <XMarkIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </li>
              );
            })
          ) : (
            <li className="px-6 py-8 text-center">
              <ShareIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No shared resources</h3>
              <p className="mt-1 text-sm text-gray-500">
                Start sharing your folders and notes with team members.
              </p>
            </li>
          )}
        </ul>
      </div>
    </div>
  );
};

export default Shared;
