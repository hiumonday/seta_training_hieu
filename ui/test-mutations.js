// Test createUser và login mutations
const testUserMutations = async () => {
  try {
    console.log('🧪 Testing createUser mutation...');
    
    // Test createUser
    const createMutation = `
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

    const createVariables = {
      input: {
        username: "testuser2025",
        email: "testuser2025@example.com",
        password: "TestPass123!",
        role: "MEMBER"
      }
    };

    const createResponse = await fetch('http://localhost:4000/users', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: createMutation,
        variables: createVariables
      }),
    });

    const createData = await createResponse.json();
    console.log('✅ CreateUser Response:', JSON.stringify(createData, null, 2));

    if (createData.data?.createUser?.success) {
      console.log('\n🧪 Testing login mutation...');
      
      // Test login
      const loginMutation = `
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

      const loginVariables = {
        input: {
          email: "testuser2025@example.com",
          password: "TestPass123!"
        }
      };

      const loginResponse = await fetch('http://localhost:4000/users', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          query: loginMutation,
          variables: loginVariables
        }),
      });

      const loginData = await loginResponse.json();
      console.log('✅ Login Response:', JSON.stringify(loginData, null, 2));
    }

  } catch (error) {
    console.error('❌ Error testing mutations:', error);
  }
};

testUserMutations();
