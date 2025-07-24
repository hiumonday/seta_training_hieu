import { ApolloClient, InMemoryCache, createHttpLink } from "@apollo/client";

// GraphQL endpoint for Node.js service
const httpLink = createHttpLink({
  uri: "http://localhost:4000/users",
});

const client = new ApolloClient({
  link: httpLink,
  cache: new InMemoryCache(),
  defaultOptions: {
    watchQuery: {
      errorPolicy: "all",
    },
    query: {
      errorPolicy: "all",
    },
  },
});

export default client;
