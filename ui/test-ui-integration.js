console.log('Testing SETA UI Authentication Flow');

// Test registration
async function testRegistration() {
  console.log('\n--- Testing Registration ---');
  
  const registerData = {
    username: 'testuser2',
    email: 'testuser2@example.com', 
    password: 'password123',
    role: 'MEMBER'
  };

  try {
    const response = await fetch('http://localhost:4000/users', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: `
          mutation CreateUser($input: CreateUserInput!) {
            createUser(input: $input) {
              code
              success
              message
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
        `,
        variables: {
          input: registerData
        }
      })
    });

    const result = await response.json();
    
    if (result.errors) {
      console.error('Registration Error:', result.errors);
      return null;
    }
    
    console.log('Registration Success:', {
      code: result.data.createUser.code,
      success: result.data.createUser.success,
      message: result.data.createUser.message,
      user: result.data.createUser.user
    });
    
    return result.data.createUser.accessToken;
  } catch (error) {
    console.error('Registration failed:', error);
    return null;
  }
}

// Test login
async function testLogin() {
  console.log('\n--- Testing Login ---');
  
  const loginData = {
    email: 'testuser2@example.com',
    password: 'password123'
  };

  try {
    const response = await fetch('http://localhost:4000/users', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: `
          mutation Login($input: UserInput!) {
            login(input: $input) {
              code
              success
              message
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
        `,
        variables: {
          input: loginData
        }
      })
    });

    const result = await response.json();
    
    if (result.errors) {
      console.error('Login Error:', result.errors);
      return null;
    }
    
    console.log('Login Success:', {
      code: result.data.login.code,
      success: result.data.login.success,
      message: result.data.login.message,
      user: result.data.login.user
    });
    
    return result.data.login.accessToken;
  } catch (error) {
    console.error('Login failed:', error);
    return null;
  }
}

// Test Go API connectivity (with token from login)
async function testGoAPI(token) {
  console.log('\n--- Testing Go API Connection ---');
  
  if (!token) {
    console.log('No token available, skipping Go API test');
    return;
  }
  
  try {
    const response = await fetch('http://localhost:8080/api/teams', {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    });
    
    if (response.ok) {
      const data = await response.json();
      console.log('Go API Response:', data);
    } else {
      console.log('Go API Status:', response.status, response.statusText);
    }
  } catch (error) {
    console.log('Go API not available:', error.message);
  }
}

// Run tests
async function runTests() {
  console.log('🚀 Starting SETA UI Integration Tests');
  
  // Test registration
  const regToken = await testRegistration();
  
  // Test login  
  const loginToken = await testLogin();
  
  // Test Go API with login token
  await testGoAPI(loginToken);
  
  console.log('\n✅ Tests completed!');
}

runTests();
