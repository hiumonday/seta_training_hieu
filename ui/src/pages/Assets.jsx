import { useState, useEffect } from 'react';
import api from '../services/api';
import {
  FolderIcon,
  DocumentIcon,
  PlusIcon,
  PencilIcon,
  TrashIcon,
  FolderPlusIcon,
} from '@heroicons/react/24/outline';

const Assets = () => {
  const [folders, setFolders] = useState([]);
  const [notes, setNotes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreateFolder, setShowCreateFolder] = useState(false);
  const [showCreateNote, setShowCreateNote] = useState(false);
  const [editingItem, setEditingItem] = useState(null);
  const [newFolderTitle, setNewFolderTitle] = useState('');
  const [newNoteData, setNewNoteData] = useState({
    title: '',
    content: '',
    folder_id: '',
  });

  useEffect(() => {
    fetchAssets();
  }, []);

  const fetchAssets = async () => {
    try {
      setLoading(true);
      const [foldersResponse, notesResponse] = await Promise.all([
        api.get('/folders'),
        api.get('/notes'),
      ]);
      setFolders(foldersResponse.data);
      setNotes(notesResponse.data);
    } catch (error) {
      console.error('Error fetching assets:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateFolder = async (e) => {
    e.preventDefault();
    if (!newFolderTitle.trim()) return;

    try {
      await api.post('/folders', { title: newFolderTitle.trim() });
      setNewFolderTitle('');
      setShowCreateFolder(false);
      fetchAssets();
    } catch (error) {
      console.error('Error creating folder:', error);
      alert('Failed to create folder');
    }
  };

  const handleCreateNote = async (e) => {
    e.preventDefault();
    if (!newNoteData.title.trim() || !newNoteData.content.trim()) return;

    try {
      const notePayload = {
        title: newNoteData.title.trim(),
        content: newNoteData.content.trim(),
      };
      
      if (newNoteData.folder_id) {
        notePayload.folder_id = parseInt(newNoteData.folder_id);
      }

      await api.post('/notes', notePayload);
      setNewNoteData({ title: '', content: '', folder_id: '' });
      setShowCreateNote(false);
      fetchAssets();
    } catch (error) {
      console.error('Error creating note:', error);
      alert('Failed to create note');
    }
  };

  const handleDeleteFolder = async (folderId) => {
    if (!confirm('Are you sure you want to delete this folder? This will also delete all notes in it.')) {
      return;
    }

    try {
      await api.delete(`/folders/${folderId}`);
      fetchAssets();
    } catch (error) {
      console.error('Error deleting folder:', error);
      alert('Failed to delete folder');
    }
  };

  const handleDeleteNote = async (noteId) => {
    if (!confirm('Are you sure you want to delete this note?')) {
      return;
    }

    try {
      await api.delete(`/notes/${noteId}`);
      fetchAssets();
    } catch (error) {
      console.error('Error deleting note:', error);
      alert('Failed to delete note');
    }
  };

  const handleUpdateFolder = async (folderId, newTitle) => {
    if (!newTitle.trim()) return;

    try {
      await api.put(`/folders/${folderId}`, { title: newTitle.trim() });
      setEditingItem(null);
      fetchAssets();
    } catch (error) {
      console.error('Error updating folder:', error);
      alert('Failed to update folder');
    }
  };

  const handleUpdateNote = async (noteId, noteData) => {
    if (!noteData.title.trim() || !noteData.content.trim()) return;

    try {
      await api.put(`/notes/${noteId}`, noteData);
      setEditingItem(null);
      fetchAssets();
    } catch (error) {
      console.error('Error updating note:', error);
      alert('Failed to update note');
    }
  };

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
          <h1 className="text-2xl font-bold text-gray-900">Assets</h1>
          <p className="mt-2 text-sm text-gray-700">
            Manage your training folders and notes.
          </p>
        </div>
        <div className="mt-4 sm:mt-0 sm:ml-16 sm:flex-none space-x-3">
          <button
            type="button"
            onClick={() => setShowCreateFolder(true)}
            className="inline-flex items-center justify-center rounded-md border border-transparent bg-indigo-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-indigo-700"
          >
            <FolderPlusIcon className="-ml-1 mr-2 h-4 w-4" />
            New Folder
          </button>
          <button
            type="button"
            onClick={() => setShowCreateNote(true)}
            className="inline-flex items-center justify-center rounded-md border border-transparent bg-green-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-green-700"
          >
            <PlusIcon className="-ml-1 mr-2 h-4 w-4" />
            New Note
          </button>
        </div>
      </div>

      {/* Create Folder Form */}
      {showCreateFolder && (
        <div className="mb-8 bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Create New Folder</h3>
          <form onSubmit={handleCreateFolder}>
            <div className="flex gap-4">
              <input
                type="text"
                value={newFolderTitle}
                onChange={(e) => setNewFolderTitle(e.target.value)}
                placeholder="Folder title"
                className="flex-1 rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                required
              />
              <button
                type="submit"
                className="bg-indigo-600 text-white px-4 py-2 rounded-md hover:bg-indigo-700"
              >
                Create
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowCreateFolder(false);
                  setNewFolderTitle('');
                }}
                className="bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Create Note Form */}
      {showCreateNote && (
        <div className="mb-8 bg-white shadow rounded-lg p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Create New Note</h3>
          <form onSubmit={handleCreateNote} className="space-y-4">
            <div>
              <input
                type="text"
                value={newNoteData.title}
                onChange={(e) => setNewNoteData({ ...newNoteData, title: e.target.value })}
                placeholder="Note title"
                className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                required
              />
            </div>
            <div>
              <select
                value={newNoteData.folder_id}
                onChange={(e) => setNewNoteData({ ...newNoteData, folder_id: e.target.value })}
                className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
              >
                <option value="">No folder (root)</option>
                {folders.map(folder => (
                  <option key={folder.id} value={folder.id}>
                    {folder.title}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <textarea
                value={newNoteData.content}
                onChange={(e) => setNewNoteData({ ...newNoteData, content: e.target.value })}
                placeholder="Note content"
                rows={6}
                className="w-full rounded-md border border-gray-300 px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                required
              />
            </div>
            <div className="flex gap-4">
              <button
                type="submit"
                className="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700"
              >
                Create Note
              </button>
              <button
                type="button"
                onClick={() => {
                  setShowCreateNote(false);
                  setNewNoteData({ title: '', content: '', folder_id: '' });
                }}
                className="bg-gray-300 text-gray-700 px-4 py-2 rounded-md hover:bg-gray-400"
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      )}

      {/* Folders Section */}
      <div className="mb-8">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Folders</h2>
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          <ul className="divide-y divide-gray-200">
            {folders.length > 0 ? (
              folders.map((folder) => (
                <li key={folder.id} className="px-6 py-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center">
                      <FolderIcon className="h-6 w-6 text-yellow-500 mr-3" />
                      {editingItem?.type === 'folder' && editingItem?.id === folder.id ? (
                        <input
                          type="text"
                          defaultValue={folder.title}
                          onBlur={(e) => handleUpdateFolder(folder.id, e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                              handleUpdateFolder(folder.id, e.target.value);
                            }
                            if (e.key === 'Escape') {
                              setEditingItem(null);
                            }
                          }}
                          className="text-lg font-medium text-gray-900 border-b-2 border-indigo-500 bg-transparent focus:outline-none"
                          autoFocus
                        />
                      ) : (
                        <h3 className="text-lg font-medium text-gray-900">{folder.title}</h3>
                      )}
                    </div>
                    <div className="flex space-x-2">
                      <button
                        onClick={() => setEditingItem({ type: 'folder', id: folder.id })}
                        className="text-indigo-600 hover:text-indigo-900"
                      >
                        <PencilIcon className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => handleDeleteFolder(folder.id)}
                        className="text-red-600 hover:text-red-900"
                      >
                        <TrashIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                  <p className="mt-1 text-sm text-gray-500">
                    Created: {new Date(folder.created_at).toLocaleDateString()}
                  </p>
                </li>
              ))
            ) : (
              <li className="px-6 py-8 text-center">
                <FolderIcon className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900">No folders</h3>
                <p className="mt-1 text-sm text-gray-500">Get started by creating a new folder.</p>
              </li>
            )}
          </ul>
        </div>
      </div>

      {/* Notes Section */}
      <div>
        <h2 className="text-xl font-semibold text-gray-900 mb-4">Notes</h2>
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          <ul className="divide-y divide-gray-200">
            {notes.length > 0 ? (
              notes.map((note) => (
                <li key={note.id} className="px-6 py-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center min-w-0 flex-1">
                      <DocumentIcon className="h-6 w-6 text-blue-500 mr-3 flex-shrink-0" />
                      <div className="min-w-0 flex-1">
                        {editingItem?.type === 'note' && editingItem?.id === note.id ? (
                          <div className="space-y-2">
                            <input
                              type="text"
                              defaultValue={note.title}
                              className="w-full text-lg font-medium text-gray-900 border-b-2 border-indigo-500 bg-transparent focus:outline-none"
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') {
                                  const textarea = e.target.nextElementSibling;
                                  textarea.focus();
                                }
                                if (e.key === 'Escape') {
                                  setEditingItem(null);
                                }
                              }}
                            />
                            <textarea
                              defaultValue={note.content}
                              rows={3}
                              className="w-full text-sm text-gray-600 border border-gray-300 rounded-md px-3 py-2 focus:border-indigo-500 focus:outline-none focus:ring-indigo-500"
                              onKeyDown={(e) => {
                                if (e.key === 'Enter' && e.ctrlKey) {
                                  const title = e.target.previousElementSibling.value;
                                  const content = e.target.value;
                                  handleUpdateNote(note.id, { title, content });
                                }
                                if (e.key === 'Escape') {
                                  setEditingItem(null);
                                }
                              }}
                            />
                            <div className="flex space-x-2">
                              <button
                                onClick={() => {
                                  const titleInput = document.querySelector(`input[defaultValue="${note.title}"]`);
                                  const contentTextarea = document.querySelector(`textarea[defaultValue="${note.content}"]`);
                                  handleUpdateNote(note.id, {
                                    title: titleInput.value,
                                    content: contentTextarea.value
                                  });
                                }}
                                className="text-xs bg-indigo-600 text-white px-2 py-1 rounded hover:bg-indigo-700"
                              >
                                Save (Ctrl+Enter)
                              </button>
                              <button
                                onClick={() => setEditingItem(null)}
                                className="text-xs bg-gray-300 text-gray-700 px-2 py-1 rounded hover:bg-gray-400"
                              >
                                Cancel
                              </button>
                            </div>
                          </div>
                        ) : (
                          <>
                            <h3 className="text-lg font-medium text-gray-900 truncate">{note.title}</h3>
                            <p className="mt-1 text-sm text-gray-600 line-clamp-2">{note.content}</p>
                            <p className="mt-1 text-xs text-gray-500">
                              {note.folder_id && (
                                <>
                                  Folder: <span className="font-medium">
                                    {folders.find(f => f.id === note.folder_id)?.title || 'Unknown'}
                                  </span> • 
                                </>
                              )}
                              Created: {new Date(note.created_at).toLocaleDateString()}
                            </p>
                          </>
                        )}
                      </div>
                    </div>
                    <div className="flex space-x-2 ml-4">
                      <button
                        onClick={() => setEditingItem({ type: 'note', id: note.id })}
                        className="text-indigo-600 hover:text-indigo-900"
                      >
                        <PencilIcon className="h-4 w-4" />
                      </button>
                      <button
                        onClick={() => handleDeleteNote(note.id)}
                        className="text-red-600 hover:text-red-900"
                      >
                        <TrashIcon className="h-4 w-4" />
                      </button>
                    </div>
                  </div>
                </li>
              ))
            ) : (
              <li className="px-6 py-8 text-center">
                <DocumentIcon className="mx-auto h-12 w-12 text-gray-400" />
                <h3 className="mt-2 text-sm font-medium text-gray-900">No notes</h3>
                <p className="mt-1 text-sm text-gray-500">Get started by creating a new note.</p>
              </li>
            )}
          </ul>
        </div>
      </div>
    </div>
  );
};

export default Assets;
