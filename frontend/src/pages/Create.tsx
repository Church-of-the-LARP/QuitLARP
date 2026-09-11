import client from "../scripts/api";

export default function Create() {

  type difficulty = "easy" | "medium" | "hard"

  async function CreateAssesment(e: FormData) {
    const title = e.get("title") as string;
    const description = e.get("description") as string;
    const difficulty = e.get("difficulty") as difficulty ;
    const timeLimitMinutes = e.get("time") as unknown as number

    try {
      const res = await client.POST("/api/v1/assessments", {
        body: { title, description, difficulty, timeLimitMinutes,  },
      });
      return res.data?.assessment ?? null;
    } catch (err) {
      console.error("Failed to load the session", err);
    }
  }

  return (
    <form action={CreateAssesment}>
      <div>
        <label>Title</label>
        <input
          type="text"
          name="title"
          className="w-90 px-3 py-2 border rounded-xl mb-3"
          placeholder="Title"
        />
      </div>
      <div>
        <label>Description</label>
        <input
          type="text"
          name="description"
          className="w-90 px-3 py-2 border rounded-xl mb-3"
          placeholder="Description"
        />
      </div>
      <div>
        <label>Difficulty</label>
        <select name="difficulty">
          <option value="hard">Hard</option>
        </select>
      </div>
      <div>
        <label>Time limit in minutes</label>
        <input
          type="number"
          name="time"
          className="w-90 px-3 py-2 border rounded-xl mb-3"
          placeholder="Time in minutes"
        />
      </div>
    </form>
  );
}
