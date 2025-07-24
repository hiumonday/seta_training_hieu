// Test to check if user service is running
const testUserService = async () => {
  try {
    console.log('Testing connection to user service...');
    
    // Test basic connection
    const response = await fetch('http://localhost:4000/users', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: `
          query {
            __typename
          }
        `
      }),
    });

    console.log('Response status:', response.status);
    console.log('Response headers:', Object.fromEntries(response.headers.entries()));
    
    if (!response.ok) {
      console.log('Response not OK. Status:', response.status);
      const text = await response.text();
      console.log('Response body:', text);
      return;
    }

    const data = await response.json();
    console.log('Success! User service is running.');
    console.log('Response:', JSON.stringify(data, null, 2));

  } catch (error) {
    console.error('Error connecting to user service:', error.message);
    
    if (error.code === 'ECONNREFUSED') {
      console.log('\n❌ User service is not running!');
      console.log('Please start the user service first:');
      console.log('1. Navigate to the user service directory');
      console.log('2. Run: npm start or node server.js');
    }
  }
};

testUserService();
