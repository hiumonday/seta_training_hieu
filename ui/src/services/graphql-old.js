import { gql } from "@apollo/client";

// Authentication mutations
export const LOGIN_MUTATION = gql`
  mutation Login($input: UserInput!) {
    login(input: $input) {
      code
      success
      message
      errors
      accessToken
      refreshToken
      user {
        userId
        username
        email
        role
        createdAt
      }
    }
  }
`;

export const REGISTER_MUTATION = gql`
  mutation CreateUser($input: CreateUserInput!) {
    createUser(input: $input) {
      code
      success
      message
      errors
      user {
        userId
        username
        email
        role
        createdAt
      }
    }
  }
`;
  }
`;

// User queries and mutations
export const GET_USERS_QUERY = gql`
  query GetUsers {
    getUsers {
      id
      username
      email
      role
      createdAt
    }
  }
`;

export const UPDATE_USER_ROLE_MUTATION = gql`
  mutation UpdateUserRole($id: ID!, $role: String!) {
    updateUserRole(id: $id, role: $role) {
      id
      username
      email
      role
    }
  }
`;

export const DELETE_USER_MUTATION = gql`
  mutation DeleteUser($id: ID!) {
    deleteUser(id: $id) {
      success
      message
    }
  }
`;

// Team queries and mutations
export const GET_TEAMS_QUERY = gql`
  query GetTeams {
    getTeams {
      id
      name
      createdAt
      users {
        id
        username
        email
        role
      }
    }
  }
`;

export const CREATE_TEAM_MUTATION = gql`
  mutation CreateTeam($name: String!) {
    createTeam(name: $name) {
      id
      name
      createdAt
    }
  }
`;

export const ADD_USER_TO_TEAM_MUTATION = gql`
  mutation AddUserToTeam($teamId: ID!, $userId: ID!) {
    addUserToTeam(teamId: $teamId, userId: $userId) {
      success
      message
    }
  }
`;

export const REMOVE_USER_FROM_TEAM_MUTATION = gql`
  mutation RemoveUserFromTeam($teamId: ID!, $userId: ID!) {
    removeUserFromTeam(teamId: $teamId, userId: $userId) {
      success
      message
    }
  }
`;
