const rosterModel = (sequelize, DataTypes) => {
  const Roster = sequelize.define(
    "Roster",
    {
      rosterId: {
        type: DataTypes.INTEGER,
        primaryKey: true,
        autoIncrement: true,
      },
      teamId: {
        type: DataTypes.INTEGER,
        references: { model: "Teams", key: "teamId" },
        onDelete: "CASCADE",
      },
      userId: {
        type: DataTypes.INTEGER,
        references: { model: "Users", key: "userId" },
        onDelete: "CASCADE",
      },
      isLeader: {
        type: DataTypes.BOOLEAN,
        defaultValue: false,
      },
    },
    {
      tableName: "Rosters",
      timestamps: false,
    }
  );

  Roster.associate = (models) => {
    // One-To-Many relationship
    Roster.belongsTo(models.User, { foreignKey: "userId", as: "user" });
    Roster.belongsTo(models.Team, { foreignKey: "teamId", as: "team" });
  };

  return Roster;
};

export default rosterModel;
