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

// User queries - Note: The schema requires role parameter for users query
export const GET_USERS_QUERY = gql`
  query GetUsers($role: UserType!) {
    users(role: $role) {
      userId
      username
      email
      role
      createdAt
    }
  }
`;

// Get all users by querying both roles
export const GET_ALL_USERS_QUERY = gql`
  query GetAllUsers {
    managers: users(role: MANAGER) {
      userId
      username
      email
      role
      createdAt
    }
    members: users(role: MEMBER) {
      userId
      username
      email
      role
      createdAt
    }
  }
`;

export const UPDATE_USER_MUTATION = gql`
  mutation UpdateUser($userId: ID!, $username: String!, $email: String!) {
    updateUser(userId: $userId, username: $username, email: $email) {
      code
      success
      message
      user {
        userId
        username
        email
        role
      }
    }
  }
`;

// Team queries and mutations
export const GET_TEAMS_QUERY = gql`
  query GetTeams($userId: ID!) {
    teams(userId: $userId) {
      teamId
      teamName
      totalManagers
      totalMembers  
      rosterCount
      managers {
        managerId
        managerName
      }
      members {
        memberId
        memberName
      }
    }
  }
`;

export const GET_MY_TEAMS_QUERY = gql`
  query GetMyTeams($userId: ID!) {
    myTeams(userId: $userId) {
      teamId
      teamName
      totalManagers
      totalMembers
      rosterCount
      managers {
        managerId
        managerName
      }
      members {
        memberId
        memberName
      }
    }
  }
`;
