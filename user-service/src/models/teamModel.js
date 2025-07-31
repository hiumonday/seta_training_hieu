const teamModel = (sequelize, DataTypes) => {
  const Team = sequelize.define(
    "Team",
    {
      teamId: {
        type: DataTypes.INTEGER,
        primaryKey: true,
        autoIncrement: true,
      },
      teamName: {
        type: DataTypes.STRING,
        allowNull: false,
        unique: true,
      },
    },
    {
      tableName: "Teams",
      timestamps: true,
    }
  );

  Team.associate = (models) => {
    // Many-To-Many relationship
    Team.belongsToMany(models.User, {
      through: models.Roster,
      foreignKey: "teamId",
      otherKey: "userId",
      as: "managers",
      scope: { role: "MANAGER" },
    });
    Team.belongsToMany(models.User, {
      through: models.Roster,
      foreignKey: "teamId",
      otherKey: "userId",
      as: "members",
      scope: { role: "MEMBER" },
    });
    // One-To-Many relationship
    Team.hasMany(models.Roster, {
      foreignKey: "teamId",
      as: "rosters",
    });
  };

  return Team;
};

export default teamModel;
